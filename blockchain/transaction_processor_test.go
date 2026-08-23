package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestApplyTransaction(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()

	initialBalance :=
		uint64(1000) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialBalance,
	); err != nil {
		t.Fatal(err)
	}

	amount :=
		uint64(100) *
			params.CoinDecimals

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	minerFee, err :=
		ApplyTransaction(
			state,
			tx,
		)

	if err != nil {
		t.Fatal(err)
	}

	expectedFee :=
		transactions.CalculateFee(
			amount,
		)

	expectedDevelopment :=
		transactions.DevelopmentFee(
			amount,
		)

	expectedMiner :=
		transactions.MiningFee(
			amount,
		)

	if minerFee != expectedMiner {
		t.Fatalf(
			"expected miner fee %d, got %d",
			expectedMiner,
			minerFee,
		)
	}

	expectedSender :=
		initialBalance -
			amount -
			expectedFee

	if state.Balance(
		sender.Address,
	) != expectedSender {

		t.Fatalf(
			"expected sender balance %d, got %d",
			expectedSender,
			state.Balance(sender.Address),
		)
	}

	if state.Balance(
		receiver.Address,
	) != amount {

		t.Fatalf(
			"expected receiver balance %d, got %d",
			amount,
			state.Balance(receiver.Address),
		)
	}

	if state.Balance(
		params.DevelopmentTreasuryAddress,
	) != expectedDevelopment {

		t.Fatalf(
			"expected development balance %d, got %d",
			expectedDevelopment,
			state.Balance(
				params.DevelopmentTreasuryAddress,
			),
		)
	}

	if state.AccountNonce(
		sender.Address,
	) != 1 {

		t.Fatalf(
			"expected sender nonce 1, got %d",
			state.AccountNonce(sender.Address),
		)
	}

	if !state.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"transaction was not marked processed",
		)
	}
}

func TestApplyTransactionInsufficientBalance(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()

	initialBalance :=
		uint64(10) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialBalance,
	); err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		uint64(100)*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	_, err =
		ApplyTransaction(
			state,
			tx,
		)

	if err == nil {
		t.Fatal(
			"expected insufficient balance error",
		)
	}

	if state.Balance(
		sender.Address,
	) != initialBalance {

		t.Fatal(
			"sender balance changed after rejected transaction",
		)
	}

	if state.Balance(
		receiver.Address,
	) != 0 {

		t.Fatal(
			"receiver balance changed after rejected transaction",
		)
	}

	if state.Balance(
		params.DevelopmentTreasuryAddress,
	) != 0 {

		t.Fatal(
			"treasury balance changed after rejected transaction",
		)
	}

	if state.AccountNonce(
		sender.Address,
	) != 0 {

		t.Fatal(
			"nonce changed after rejected transaction",
		)
	}

	if state.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"rejected transaction was marked processed",
		)
	}
}

func TestApplyTransactionUsesPermanentTreasury(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()

	if state.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatalf(
			"expected permanent treasury %s, got %s",
			params.DevelopmentTreasuryAddress,
			state.DevelopmentAddress(),
		)
	}

	initialBalance :=
		uint64(1000) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialBalance,
	); err != nil {
		t.Fatal(err)
	}

	amount :=
		uint64(100) *
			params.CoinDecimals

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err :=
		ApplyTransaction(
			state,
			tx,
		); err != nil {

		t.Fatal(err)
	}

	expectedDevelopment :=
		transactions.DevelopmentFee(
			amount,
		)

	if state.Balance(
		params.DevelopmentTreasuryAddress,
	) != expectedDevelopment {

		t.Fatalf(
			"expected permanent treasury balance %d, got %d",
			expectedDevelopment,
			state.Balance(
				params.DevelopmentTreasuryAddress,
			),
		)
	}
}
