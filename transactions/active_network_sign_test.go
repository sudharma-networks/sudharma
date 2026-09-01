package transactions

import (
	"testing"

	"github.com/sudharma-networks/sudharma/networkcontext"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestSignUsesActiveProcessNetwork(t *testing.T) {
	networkcontext.Set(params.NetworkMainnet)
	t.Cleanup(networkcontext.Reset)

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	tx := NewTransaction(sender.Address, receiver.Address, params.MinTransferAmount, 1)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if !tx.VerifyForNetwork(params.NetworkMainnet) {
		t.Fatal("default signer did not bind signature to active mainnet context")
	}
	if tx.VerifyForNetwork(params.NetworkPublicTestnet) {
		t.Fatal("active-mainnet signature was accepted on public testnet")
	}
}

func TestSignDefaultsToPublicTestnetContext(t *testing.T) {
	networkcontext.Reset()
	t.Cleanup(networkcontext.Reset)

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	tx := NewTransaction(sender.Address, receiver.Address, params.MinTransferAmount, 1)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if !tx.VerifyForNetwork(params.NetworkPublicTestnet) {
		t.Fatal("default signer did not use public-testnet context")
	}
}
