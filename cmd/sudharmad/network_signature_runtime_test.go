package main

import (
	"testing"

	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestDaemonActiveNetworkBindsConvenienceTransactionSigner(t *testing.T) {
	p2p.SetLocalNetworkID(params.NetworkMainnet)
	t.Cleanup(p2p.ResetLocalNetworkIDForTests)

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(sender.Address, receiver.Address, params.MinTransferAmount, 1)
	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if !tx.VerifyForNetwork(params.NetworkMainnet) {
		t.Fatal("daemon convenience signer did not bind to the active mainnet identity")
	}
	if tx.VerifyForNetwork(params.NetworkPublicTestnet) {
		t.Fatal("mainnet-bound daemon signature replayed on public testnet")
	}
}
