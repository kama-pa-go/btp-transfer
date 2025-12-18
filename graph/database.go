package graph

import (
	"context"
	"fmt"
)

// ExecuteTransfer does not contain aPI logic.
// Api logic connected to Transfer operation can be found in schema.resolvers.go file
func (r *Resolver) ExecuteTransfer(ctx context.Context, fromAddress, toAddress string, amount int32) (int32, error) {
	tx, err := r.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Defer function to handle rollback in case of panic or error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			// if there was no errors commit changes
			err = tx.Commit()
		}
	}()

	// -- Prevention of deadlocks: --
	//Sort addresses: always put them in alphabetical order
	firstLock := fromAddress
	secondLock := toAddress
	if fromAddress > toAddress {
		firstLock = toAddress
		secondLock = fromAddress
	}

	// Block first address
	// Ignore "haven't found" error-receiver may have been not created yet
	_, err = tx.ExecContext(ctx, "SELECT 1 FROM wallets WHERE address = $1 FOR UPDATE", firstLock)

	// Block second address
	// (but only if is different from the first one)
	if firstLock != secondLock {
		_, err = tx.ExecContext(ctx, "SELECT 1 FROM wallets WHERE address = $1 FOR UPDATE", secondLock)
	}

	// Downland sender's balance
	// FOR UPDATE is not necessary but won't hurt either
	// Other processes are frozen until it's done
	var currentBalance int32
	row := tx.QueryRowContext(ctx, "SELECT balance FROM wallets WHERE address = $1", fromAddress)
	err = row.Scan(&currentBalance)

	// -- Deadlocks prevented --

	// Check if balance is sufficient
	if currentBalance < amount {
		err = fmt.Errorf("insufficient balance")
		return 0, err
	}

	// Subtract means from sender
	_, err = tx.ExecContext(ctx, "UPDATE wallets SET balance = balance - $1 WHERE address = $2", amount, fromAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to deduct funds: %w", err)
	}

	// Add means to reciver
	// UPSERT: if wallet exists update, else: make a new one
	_, err = tx.ExecContext(ctx, `
		INSERT INTO wallets (address, balance) VALUES ($1, $2)
		ON CONFLICT (address) DO UPDATE SET balance = wallets.balance + $2
	`, toAddress, amount)
	if err != nil {
		return 0, fmt.Errorf("failed to add funds to receiver: %w", err)
	}

	// Return new balance
	return currentBalance - amount, nil
}
