package transactions

import (
	"testing"

	"github.com/sudharma-networks/sudharma/wallet"
)

func TestDifferentNoncesProduceDifferentTransactionIDs(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	amount := uint64(100 * 100_000_000)

	tx1 := NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		1,
	)

	tx2 := NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		2,
	)

	if tx1.ID == tx2.ID {
		t.Fatal(
			"different nonces produced identical transaction IDs",
		)
	}
}

func TestNonceTamperingInvalidatesSignature(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := NewTransaction(
		sender.Address,
		receiver.Address,
		100*100_000_000,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if !tx.Verify() {
		t.Fatal("original transaction should be valid")
	}

	// Attacker changes the transaction nonce.
	tx.Nonce = 2

	if tx.Verify() {
		t.Fatal(
			"transaction with tampered nonce was accepted",
		)
	}
}
