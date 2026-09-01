package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestDecodeTransactionForNetworkUsesExplicitSignatureDomain(t *testing.T) {
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
		params.MinTransferAmount,
		1,
	)
	if err := tx.SignForNetwork(sender, params.NetworkMainnet); err != nil {
		t.Fatal(err)
	}

	encoded, err := NewTransactionMessage(tx)
	if err != nil {
		t.Fatal(err)
	}
	message, err := DecodeMessage(encoded)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeTransactionForNetwork(message, params.NetworkMainnet)
	if err != nil {
		t.Fatalf("matching mainnet transaction was rejected: %v", err)
	}
	if decoded.ID != tx.ID {
		t.Fatalf("decoded transaction id = %q, want %q", decoded.ID, tx.ID)
	}
	if _, err := DecodeTransaction(message); err == nil {
		t.Fatal("mainnet-bound transaction unexpectedly passed default testnet decoder")
	}
}
