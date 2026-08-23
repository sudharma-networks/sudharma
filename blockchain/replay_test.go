package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionReplayRejected(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	development, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()
	state.SetDevelopmentAddress(development.Address)

	initialBalance := uint64(1000 * 100_000_000)
	state.Credit(sender.Address, initialBalance)

	amount := uint64(100 * 100_000_000)

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	_, err = ApplyTransaction(state, tx)
	if err != nil {
		t.Fatalf(
			"first transaction should succeed: %v",
			err,
		)
	}

	if state.AccountNonce(sender.Address) != 1 {
		t.Fatalf(
			"expected nonce 1, got %d",
			state.AccountNonce(sender.Address),
		)
	}

	senderAfterFirst := state.Balance(sender.Address)
	receiverAfterFirst := state.Balance(receiver.Address)
	developmentAfterFirst := state.Balance(development.Address)

	// Submit exactly the same signed transaction again.
	_, err = ApplyTransaction(state, tx)

	if err == nil {
		t.Fatal("replayed transaction was accepted")
	}

	if state.Balance(sender.Address) != senderAfterFirst {
		t.Fatal("sender balance changed after replay")
	}

	if state.Balance(receiver.Address) != receiverAfterFirst {
		t.Fatal("receiver balance changed after replay")
	}

	if state.Balance(development.Address) != developmentAfterFirst {
		t.Fatal("development balance changed after replay")
	}

	if state.AccountNonce(sender.Address) != 1 {
		t.Fatal("sender nonce changed after replay")
	}

	if !state.IsTransactionProcessed(tx.ID) {
		t.Fatal("transaction was not recorded as processed")
	}
}
