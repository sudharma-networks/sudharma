package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestForgedTransactionRejected(t *testing.T) {
	victim, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	attacker, err := wallet.NewWallet()
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

	initialVictimBalance := uint64(500 * 100_000_000)
	state.Credit(victim.Address, initialVictimBalance)

	// Attacker creates a transaction claiming the victim's address.
	tx := transactions.NewTransaction(
		victim.Address,
		receiver.Address,
		100*100_000_000,
	)

	// The attacker's wallet must not be able to sign for the victim.
	if err := tx.Sign(attacker); err == nil {
		t.Fatal("attacker should not be able to sign for victim")
	}

	_, err = ApplyTransaction(state, tx)

	if err == nil {
		t.Fatal("forged transaction was accepted")
	}

	if state.Balance(victim.Address) != initialVictimBalance {
		t.Fatalf(
			"victim balance changed: expected %d, got %d",
			initialVictimBalance,
			state.Balance(victim.Address),
		)
	}

	if state.Balance(receiver.Address) != 0 {
		t.Fatalf(
			"receiver got funds from forged transaction: %d",
			state.Balance(receiver.Address),
		)
	}

	if state.Balance(development.Address) != 0 {
		t.Fatalf(
			"development treasury changed after forged transaction: %d",
			state.Balance(development.Address),
		)
	}
}
