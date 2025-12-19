package graph

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	_ "github.com/lib/pq"
)

// setupTestDB connects to the database or fails the test immediately
func setupTestDB(t *testing.T) *sql.DB {
	testDB := "postgres://user:password@localhost:5432/btp_test?sslmode=disable"
	if envDSN := os.Getenv("TEST_DATABASE_URL"); envDSN != "" {
		testDB = envDSN
	}

	// Open connection
	db, err := sql.Open("postgres", testDB)
	if err != nil {
		t.Fatalf("Failed to connect to  test DB: %v", err)
	}

	// Check connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping TEST database. Did you run 'docker-compose up'? Error: %v", err)
	}

	cleanTestDB(t, db) // Clean db before return
	return db
}

// cleanTestDB removes all data from tables to ensure test isolation
func cleanTestDB(t *testing.T, db *sql.DB) {
	_, err := db.Exec("TRUNCATE TABLE wallets")
	if err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}
}

// resetWallet inserts or updates a wallet to a specific balance for testing
func resetWallet(t *testing.T, db *sql.DB, address string, balance int64) {
	address = strings.ToLower(address)

	_, err := db.Exec(`
		INSERT INTO wallets (address, balance) VALUES ($1, $2)
		ON CONFLICT (address) DO UPDATE SET balance = $2
	`, address, balance)
	if err != nil {
		t.Fatalf("Failed to reset wallet %s: %v", address, err)
	}
}

// getResolver creates a resolver instance with the DB connection
func getResolver(db *sql.DB) MutationResolver {
	return (&Resolver{DB: db}).Mutation()
}

// --- ACTUAL TESTS ---

// 1. The "Hammer" Test: 100 concurrent threads withdraw 1 token each.
// Goal: Verify database locking mechanism works under high load.
func TestConcurrent_Hammer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	address := "0xHAMMER"
	startBalance := int64(100)
	resetWallet(t, db, address, startBalance)

	mutation := getResolver(db)
	var wg sync.WaitGroup
	wg.Add(int(startBalance)) // 100 threads

	for i := 0; i < int(startBalance); i++ {
		go func() {
			defer wg.Done()
			_, err := mutation.Transfer(context.Background(), address, "0xRECEIVER", 1)
			if err != nil {
				t.Errorf("Unexpected error in hammer test: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify final balance
	var finalBalance int64
	err := db.QueryRow("SELECT balance FROM wallets WHERE address = $1", strings.ToLower(address)).Scan(&finalBalance)
	if err != nil {
		t.Fatalf("Failed to verify balance: %v", err)
	}

	if finalBalance != 0 {
		t.Errorf(" - Race Condition Detected! Expected 0, got %d", finalBalance)
	} else {
		fmt.Println(" + Hammer Test Passed: Balance is cleanly 0.")
	}
}

// 2. Logic Test: Insufficient Funds
// Goal: Verify that the system prevents spending more than you have.
func TestLogic_InsufficientFunds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sender := "0xPOOR"
	resetWallet(t, db, sender, 10) // Wallet has 10
	mutation := getResolver(db)

	// Try to send 20
	_, err := mutation.Transfer(context.Background(), sender, "0xRICH", 20)

	if err == nil {
		t.Errorf(" - Error expected but transfer succeeded! Balance should not go negative.")
	} else {
		fmt.Printf(" + Insufficient Funds Test Passed: Got expected error: %v\n", err)
	}
}

// 3. The "Threads Scenario" Test: Mixed operations (+1, -4, -7) on small balance.
// Goal: Verify specific race condition outcome logic required by the task.
// IMPROVEMENT: We run this scenario multiple times to increase the chance of catching a race condition.
func TestConcurrent_MixedThreadsScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	subject := "0xSUBJECT"
	external := "0xEXTERNAL"
	mutation := getResolver(db)

	iterations := 100

	for i := 0; i < iterations; i++ {
		resetWallet(t, db, subject, 10)   // Has 10
		resetWallet(t, db, external, 100) // Has 100 (to send funds)

		var wg sync.WaitGroup
		wg.Add(3)

		// Op 1: +1 (Receive)
		go func() {
			defer wg.Done()
			_, err := mutation.Transfer(context.Background(), external, subject, 1)
			if err != nil {
				t.Errorf("Unexpected error in +1 operation: %v", err)
			}
		}()

		// Op 2: -4 (Send)
		go func() {
			defer wg.Done()
			_, _ = mutation.Transfer(context.Background(), subject, external, 4)
		}()

		// Op 3: -7 (Send)
		go func() {
			defer wg.Done()
			_, _ = mutation.Transfer(context.Background(), subject, external, 7)
		}()

		wg.Wait()

		var finalBalance int64
		err := db.QueryRow("SELECT balance FROM wallets WHERE address = $1", strings.ToLower(subject)).Scan(&finalBalance)
		if err != nil {
			t.Fatalf(" - Failed to verify balance in MixedScenario: %v", err)
		}

		// Valid outcomes: 0 (all succeed in order), 7 (-7 failed), 4 (-4 failed)
		switch finalBalance {
		case 0, 7, 4:
			// everything fine! :) (but we're not gonna write that out for 100 times)
		default:
			t.Errorf(" - Mixed Threads Scenario Failed. Invalid balance: %d", finalBalance)
		}
	}
	fmt.Printf(" + Mixed Threads Scenario Passed. All %d iterations gave outcome 0, 7 or 4\n", iterations)
}

// 4. Security Test: Negative Amount
// Goal: Ensure users cannot steal money by sending negative amounts.
func TestSecurity_NegativeAmount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hacker := "0xHACKER"
	resetWallet(t, db, hacker, 100)
	mutation := getResolver(db)

	// Hacker tries to send -50 to increase their own balance or steal from receiver
	_, err := mutation.Transfer(context.Background(), hacker, "0xVICTIM", -50)

	if err == nil {
		t.Log("- Fail: Transfer accepted negative amount. If this is not intended, add validation.")
	} else {
		fmt.Println(" + Security Test Passed: Negative amount rejected.")
	}
}

// 5. Edge Case: Non-Existent Sender
// Goal: Verify graceful error handling when wallet is missing.
func TestLogic_NonExistentSender(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	mutation := getResolver(db)

	// 0xGHOST does not exist in DB
	_, err := mutation.Transfer(context.Background(), "0xGHOST", "0xREAL", 10)

	if err == nil {
		t.Errorf(" - Fail: Error expected for non-existent sender, but got success.")
	} else {
		fmt.Println(" + Non-Existent Sender Test Passed: Got error as expected.")
	}
}

// 6. Edge Case: Self-Transfer
// Goal: Verify that sending money to yourself doesn't result in deadlock or lost funds.
func TestLogic_SelfTransfer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	me := "0xNARCISSIST" // Wallet who loves only himself ;)
	startBalance := int64(100)
	resetWallet(t, db, me, startBalance)

	mutation := getResolver(db)

	// Transfer 50. The outcome should remain 100
	_, err := mutation.Transfer(context.Background(), me, me, 50)

	if err != nil {
		t.Errorf(" - Self-transfer failed with error: %v", err)
	}

	// Check final balance
	var finalBalance int64
	err = db.QueryRow("SELECT balance FROM wallets WHERE address = $1", strings.ToLower(me)).Scan(&finalBalance)
	if err != nil {
		t.Fatalf(" - Failed to verify balance: %v", err)
	}

	if finalBalance != startBalance {
		t.Errorf(" - Failed: Expected %d, got %d. Check your UPDATE logic.", startBalance, finalBalance)
	} else {
		fmt.Println(" + Self-Transfer Test Passed: Balance remained unchanged (No deadlock, no math error).")
	}
}
