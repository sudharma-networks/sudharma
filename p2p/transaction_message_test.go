package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionMessageEncodeDecode(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	data, err := NewTransactionMessage(tx)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	decodedTx, err := DecodeTransaction(message)
	if err != nil {
		t.Fatal(err)
	}

	if decodedTx.ID != tx.ID {
		t.Fatalf(
			"expected transaction ID %s, got %s",
			tx.ID,
			decodedTx.ID,
		)
	}

	if decodedTx.From != tx.From {
		t.Fatal(
			"decoded sender does not match",
		)
	}

	if decodedTx.To != tx.To {
		t.Fatal(
			"decoded receiver does not match",
		)
	}

	if decodedTx.Amount != tx.Amount {
		t.Fatal(
			"decoded amount does not match",
		)
	}
}

func TestTamperedTransactionMessageRejected(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*100_000_000,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	// Tamper after signing.
	tx.Amount = 1000 * 100_000_000

	data, err := NewTransactionMessage(tx)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodeTransaction(message); err == nil {
		t.Fatal(
			"tampered transaction message was accepted",
		)
	}
}

func TestNilTransactionMessageRejected(t *testing.T) {
	if _, err := NewTransactionMessage(nil); err == nil {
		t.Fatal(
			"nil transaction message was accepted",
		)
	}
}
