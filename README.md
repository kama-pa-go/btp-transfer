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
* **Decision:** The API uses `int32` for token amounts.
* **Reasoning:** While financial systems and databases (`BIGINT` in PostgreSQL) typically use 64-bit integers or arbitrary-precision strings to handle large token supplies, `int32` was chosen here for simplicity and compatibility with the default GraphQL scalar mapping in the generated Go code. It is sufficient for the initial requirement of 1,000,000 tokens.

### 2. Automatic Wallet Creation (Implicit Registration)
* **Decision:** If a transfer is made to a non-existent `to_address`, the system automatically creates that wallet using an `UPSERT` strategy (`INSERT ... ON CONFLICT`).
* **Reasoning:** The assignment restricted adding "additional functionality" (e.g., a `CreateWallet` mutation). Therefore, wallet creation is implicit during the first transfer.
* **Risk:** In a production environment, this is considered a security risk (typos in addresses lead to lost funds). However, it was a necessary compromise to fulfill the requirements without expanding the API surface.

### 3. Case Insensitivity
* **Decision:** All addresses are normalized to lowercase using `strings.ToLower()` before processing.
* **Reasoning:** Ensures that `0xABC` and `0xabc` are treated as the same wallet, consistent with common blockchain address standards.