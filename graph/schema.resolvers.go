package graph

import (
	"context"
	"fmt"
	"strings"
)

// Transfer is the resolver for the transfer field.
// In a case of any error whole transaction is recalled
func (r *mutationResolver) Transfer(ctx context.Context, fromAddress string, toAddress string, amount int32) (int32, error) {
	// Assure that 0xABC and 0xabc are pointing to the same address
	fromAddress = strings.ToLower(fromAddress)
	toAddress = strings.ToLower(toAddress)

	// Delegate operations on data to database.go
	return r.ExecuteTransfer(ctx, fromAddress, toAddress, amount)
}

// Dummy is the resolver for the dummy field.
func (r *queryResolver) Dummy(ctx context.Context) (*string, error) {
	panic(fmt.Errorf("not implemented: Dummy - dummy"))
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
