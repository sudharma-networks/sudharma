package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/transactions"
)

func ApplyTransaction(
	state *State,
	tx *transactions.Transaction,
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

	if !tx.Verify() {
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

	if developmentFee+minerFee != tx.Fee {
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

	state.Credit(
		tx.To,
		tx.Amount,
	)

	state.Credit(
		developmentAddress,
		developmentFee,
	)

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
