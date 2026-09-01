package transactions

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestNetworkBoundSignatureVerifiesOnMatchingNetwork(t *testing.T) {
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
	if err := tx.SignForNetwork(sender, params.NetworkPublicTestnet); err != nil {
		t.Fatal(err)
	}
	if !tx.VerifyForNetwork(params.NetworkPublicTestnet) {
		t.Fatal("network-bound signature rejected on matching network")
	}
}

func TestCrossNetworkSignatureReplayRejected(t *testing.T) {
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
	if err := tx.SignForNetwork(sender, params.NetworkPublicTestnet); err != nil {
		t.Fatal(err)
	}

	if tx.VerifyForNetwork(params.NetworkMainnet) {
		t.Fatal("testnet signature was accepted on mainnet")
	}
}

func TestLegacySignatureStillValidOnPublicTestnet(t *testing.T) {
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
	tx.PublicKey = append([]byte(nil), sender.PublicKey...)

	message, err := SigningMessage(
		SignatureDomainLegacy,
		params.NetworkPublicTestnet,
		tx.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := sender.Sign(message)
	if err != nil {
		t.Fatal(err)
	}
	tx.Signature = signature

	if !tx.VerifyForNetwork(params.NetworkPublicTestnet) {
		t.Fatal("legacy signature rejected on public testnet")
	}
	if tx.VerifyForNetwork(params.NetworkMainnet) {
		t.Fatal("legacy signature accepted on mainnet")
	}
}

func TestMainnetRequiresNetworkBoundSignature(t *testing.T) {
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
	tx.PublicKey = append([]byte(nil), sender.PublicKey...)

	message, err := SigningMessage(
		SignatureDomainLegacy,
		params.NetworkMainnet,
		tx.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := sender.Sign(message)
	if err != nil {
		t.Fatal(err)
	}
	tx.Signature = signature

	if tx.VerifyForNetwork(params.NetworkMainnet) {
		t.Fatal("legacy signature accepted on mainnet")
	}
}

func TestSigningMessageRejectsUnknownDomain(t *testing.T) {
	if _, err := SigningMessage(99, params.NetworkPublicTestnet, "abc"); err == nil {
		t.Fatal("expected unknown domain to be rejected")
	}
}
