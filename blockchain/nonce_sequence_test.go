package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestNonceSequence(t *testing.T) {
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

	state.Credit(
		sender.Address,
		1000*100_000_000,
	)

	// Nonce 1 must succeed.
	tx1 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		1,
	)

	if err := tx1.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyTransaction(state, tx1); err != nil {
		t.Fatalf("nonce 1 should succeed: %v", err)
	}

	if state.AccountNonce(sender.Address) != 1 {
		t.Fatalf(
			"expected account nonce 1, got %d",
			state.AccountNonce(sender.Address),
		)
	}

	// Nonce 2 must succeed.
	tx2 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		2,
	)

	if err := tx2.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyTransaction(state, tx2); err != nil {
		t.Fatalf("nonce 2 should succeed: %v", err)
	}

	if state.AccountNonce(sender.Address) != 2 {
		t.Fatalf(
			"expected account nonce 2, got %d",
			state.AccountNonce(sender.Address),
		)
	}
}

func TestReusedNonceRejected(t *testing.T) {
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

	state.Credit(
		sender.Address,
		1000*100_000_000,
	)

	tx1 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		1,
	)

	if err := tx1.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyTransaction(state, tx1); err != nil {
		t.Fatal(err)
	}

	// Different transaction, but incorrectly reuses nonce 1.
	txReuse := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		20*100_000_000,
		1,
	)

	if err := txReuse.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyTransaction(state, txReuse); err == nil {
		t.Fatal("reused nonce was accepted")
	}

	if state.AccountNonce(sender.Address) != 1 {
		t.Fatal("account nonce changed after reused nonce")
	}
}

func TestSkippedNonceRejected(t *testing.T) {
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

	state.Credit(
		sender.Address,
		1000*100_000_000,
	)

	tx1 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		1,
	)

	if err := tx1.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyTransaction(state, tx1); err != nil {
		t.Fatal(err)
	}

	// Expected nonce is now 2,
	// but sender tries to jump directly to 4.
	tx4 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		4,
	)

	if err := tx4.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyTransaction(state, tx4); err == nil {
		t.Fatal("skipped nonce was accepted")
	}

	if state.AccountNonce(sender.Address) != 1 {
		t.Fatal("account nonce changed after skipped nonce")
	}
}
