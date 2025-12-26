> **Note:** This project was developed as a solution to the recruitment task for **Beside The Park** company.

# BTP Token Transfer API

A GraphQL API implemented in Go for transferring BTP tokens between wallets. The system ensures data consistency and handles race conditions using PostgreSQL transactions with pessimistic locking (`SELECT ... FOR UPDATE`).

This project assumes an ERC20-like token environment where wallets are identified by their addresses.

## How to Run

### Prerequisites
* Docker & Docker Compose


### Start the Application
The easiest way to run the database and the backend server is to use Docker Compose:

```bash
docker-compose up --build
```

Once started:
* Open in browser *GraphQL Playground:* `http://localhost:8080/`
* For API check *API Endpoint:* `http://localhost:8080/query`

**The database is automatically seeded with a genesis wallet (`0x00...00`) holding 1,000,000 tokens.**

---

## Testing

The project includes a comprehensive suite of integration tests covering concurrency and edge cases.

### Running Automated Tests
To run the tests (ensure the database is running via Docker first):

```bash
go test -v ./graph/...
```

### Covered Scenarios
* **Race Conditions:** Simulates concurrent transfers from a single wallet to ensure atomic balance updates (including high-concurrency "hammer" tests).
* **Mixed Operations:** Handles simultaneous `+` and `-` operations to verify transaction isolation.
* **Edge Cases:** Insufficient funds, negative amounts, self-transfers, non-existent senders.

---

### Database Initialization Strategy (Updated)

The project structure has been refactored to separate schema definitions (`schema.sql`) from data seeding (`init.sql`).

**For Automated Tests:**
The test suite now uses a **self-contained setup** (via `TestMain`). It automatically:
1.  Creates the `btp_test` database if it doesn't exist.
2.  Applies the latest structure from `schema.sql`.
3.  Cleans the state before every test.

**Why this change?**
Previously, updating the test database schema required manually resetting Docker volumes (`docker-compose down -v`), because PostgreSQL containers only execute initialization scripts when the data directory is empty. While resetting volumes is still a valid method to enforce schema updates, the new approach eliminates this manual step for testing, ensuring that `go test` always runs against the current code version regardless of the local Docker state.

**Note for Development:**
For the main application (non-test environment), if you modify `schema.sql` or `init.sql`, you still need to reset volumes to see changes:
```bash
docker-compose down -v
docker-compose up --build

--

## API Usage Example

To transfer tokens, execute the following mutation in the GraphQL Playground:

```graphql
mutation {
  transfer(
    from_address: "0x0000000000000000000000000000000000000000",
    to_address: "0x123abc",
    amount: 100
  )
}
```
Feel free to enter any amount (within the 0 to max int32 range) and a to_address (if it doesn't exist, a new one will be created). 
Ensure that the from_address wallet already exists (initially, only the wallet with address "0x0000000000000000000000000000000000000000" exists).

**Response:**
Returns the updated balance of the `from_address`.

---

## Design Decisions & Trade-offs

During the implementation, several architectural compromises were made to satisfy the specific requirements of the assignment while keeping the codebase simple.

### 1. Data Types (`int32` vs `BigInt`)
* **Decision:** The API uses `int64` for token amounts.
* **Reasoning:** Financial systems require precision and range beyond standard 32-bit integers. We migrated from Int32 to Int64 (mapped to PostgreSQL BIGINT) to ensure scalability and adhere to industry standards, overcoming the default GraphQL Int limitations.

### 2. Automatic Wallet Creation (Implicit Registration)
* **Decision:** If a transfer is made to a non-existent `to_address`, the system automatically creates that wallet using an `UPSERT` strategy (`INSERT ... ON CONFLICT`).
* **Reasoning:** The assignment restricted adding "additional functionality" (e.g., a `CreateWallet` mutation). Therefore, wallet creation is implicit during the first transfer.
* **Risk:** In a production environment, this is considered a security risk (typos in addresses lead to lost funds). However, it was a necessary compromise to fulfill the requirements without expanding the API surface. Another risk is possibility of races in a case when two processes want to send mony to tha same new adress.

#### Prevention of second risk:  Race Condition Prevention
* **Decision:** Implemented a "Pre-initialization" (Upsert) step before Locking.
* **Reasoning:** Standard `SELECT ... FOR UPDATE` does not lock rows that do not exist yet. To prevent race conditions when creating new wallets under heavy load, the system performs an `INSERT ... ON CONFLICT DO NOTHING` for the receiver *before* attempting to lock the rows. This guarantees that locks are always applied to existing records.

### 3. Case Insensitivity
* **Decision:** All addresses are normalized to lowercase using `strings.ToLower()` before processing.
* **Reasoning:** Ensures that `0xABC` and `0xabc` are treated as the same wallet, consistent with common blockchain address standards.

### 4. Deadlock Prevention (Deterministic Locking)
* **Decision:** Before processing a transfer, the system locks both the sender and receiver rows in the database using a strict lexicographical order (based on address strings).
* **Reasoning:** In high-concurrency scenarios, simultaneous transfers between two wallets in opposite directions (A->B and B->A) can cause database deadlocks. By enforcing a global locking order (always lock the "smaller" address first), the system prevents circular dependencies, ensuring thread safety without relying on database retries.

### 5. Transaction Safety (Explicit Commit)
* **Decision:** Transactions are committed explicitly at the end of the operation, not in a `defer` block.
* **Reasoning:** Relying on deferred commits can lead to "phantom success" states where the function returns success, but the commit fails silently afterwards. Explicit commits ensure that any database failure is caught and reported to the user.

### 6. Input Validation & Security
* **Decision:** Strict server-side validation rejects negative amounts and zero-value transfers.
* **Reasoning:** This prevents potential exploits (e.g., stealing funds via negative transfers) and database spam. Validation logic is placed at the entry point of the service layer.

### 7. Self-Transfer Optimization
* **Decision:** Transfers where `from_address` equals `to_address` bypass the heavy transaction logic.
* **Reasoning:** Since the net balance change is zero, opening a transaction and locking rows is unnecessary overhead. These requests are handled by a lightweight read-only check.
