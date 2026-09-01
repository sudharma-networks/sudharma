package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

// ApplyTransaction validates and applies one transaction atomically under the
// default public-testnet network signature domain.
func ApplyTransaction(
	state *State,
	tx *transactions.Transaction,
) (uint64, error) {
	return ApplyTransactionFor(state, tx, params.DefaultNetwork)
}

// ApplyTransactionFor validates and applies one transaction atomically for an
// explicit network identity. All mutations are performed against a private
// state clone first. The caller's state is replaced only after every debit,
// credit, nonce and replay marker operation succeeds.
func ApplyTransactionFor(
	state *State,
	tx *transactions.Transaction,
	network params.NetworkID,
) (uint64, error) {
	if state == nil {
		return 0, fmt.Errorf("state cannot be nil")
	}

	workingState := state.Clone()
	minerFee, err := applyTransactionMutating(workingState, tx, network)
	if err != nil {
		return 0, err
	}

	if err := state.ReplaceWith(workingState); err != nil {
		return 0, err
	}

	return minerFee, nil
}

// applyTransactionMutating applies a transaction to an isolated state object.
// Callers must not pass shared live state unless they intentionally accept
// partial mutation on error. ApplyTransaction is the public fail-closed entry
// point and always supplies a private clone.
func applyTransactionMutating(
	state *State,
	tx *transactions.Transaction,
	network params.NetworkID,
) (uint64, error) {
	if state == nil {
		return 0, fmt.Errorf("state cannot be nil")
	}

	if tx == nil {
		return 0, fmt.Errorf(
			"transaction cannot be nil",
		)
	}

	if tx.ID == "" {
		return 0, fmt.Errorf(
			"transaction ID cannot be empty",
		)
	}

	if state.IsTransactionProcessed(tx.ID) {
		return 0, fmt.Errorf(
			"transaction already processed: %s",
			tx.ID,
		)
	}

	if tx.From == "" {
		return 0, fmt.Errorf(
			"transaction sender cannot be empty",
		)
	}

	if tx.To == "" {
		return 0, fmt.Errorf(
			"transaction receiver cannot be empty",
		)
	}

	if tx.Amount == 0 {
		return 0, fmt.Errorf(
			"transaction amount must be greater than zero",
		)
	}

	if !tx.VerifyForNetwork(network) {
		return 0, fmt.Errorf(
			"invalid transaction signature",
		)
	}

	// Enforce sequential account nonce.
	expectedNonce, err :=
		state.ExpectedNonce(tx.From)

	if err != nil {
		return 0, err
	}

	if tx.Nonce != expectedNonce {
		return 0, fmt.Errorf(
			"invalid transaction nonce: expected %d, got %d",
			expectedNonce,
			tx.Nonce,
		)
	}

	expectedFee :=
		transactions.CalculateFee(tx.Amount)

	if tx.Fee != expectedFee {
		return 0, fmt.Errorf(
			"invalid transaction fee",
		)
	}

	developmentFee :=
		transactions.DevelopmentFee(tx.Amount)

	minerFee :=
		transactions.MiningFee(tx.Amount)

	if developmentFee > tx.Fee || minerFee != tx.Fee-developmentFee {
		return 0, fmt.Errorf(
			"invalid fee distribution",
		)
	}

	if tx.Amount > ^uint64(0)-tx.Fee {
		return 0, fmt.Errorf(
			"transaction amount overflow",
		)
	}

	totalCost := tx.Amount + tx.Fee

	if state.Balance(tx.From) < totalCost {
		return 0, fmt.Errorf(
			"insufficient balance: have %d, need %d",
			state.Balance(tx.From),
			totalCost,
		)
	}

	developmentAddress :=
		state.DevelopmentAddress()

	if developmentAddress == "" {
		return 0, fmt.Errorf(
			"development treasury address is not configured",
		)
	}

	if err := state.Debit(
		tx.From,
		totalCost,
	); err != nil {
		return 0, err
	}

	if err := state.Credit(
		tx.To,
		tx.Amount,
	); err != nil {
		return 0, fmt.Errorf("receiver credit failed: %w", err)
	}

	if err := state.Credit(
		developmentAddress,
		developmentFee,
	); err != nil {
		return 0, fmt.Errorf("development treasury credit failed: %w", err)
	}

	if err := state.SetAccountNonce(
		tx.From,
		tx.Nonce,
	); err != nil {
		return 0, err
	}

	if err := state.MarkTransactionProcessed(
		tx.ID,
	); err != nil {
		return 0, err
	}

	return minerFee, nil
}
