package transactions

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMainnetSignatureReplayRejectedOnPublicTestnet(t *testing.T) {
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
		1000*params.CoinDecimals,
		0,
	)
	if err := tx.SignForNetwork(sender, params.NetworkMainnet); err != nil {
		t.Fatal(err)
	}
	if !tx.VerifyForNetwork(params.NetworkMainnet) {
		t.Fatal("mainnet signature rejected on matching mainnet domain")
	}
	if tx.VerifyForNetwork(params.NetworkPublicTestnet) {
		t.Fatal("mainnet signature was accepted on public testnet")
	}
}
