package transactions

import (
	"testing"

	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionFees(t *testing.T) {
	amount := uint64(1000 * 100_000_000)

	totalFee := CalculateFee(amount)
	developmentFee := DevelopmentFee(amount)
	miningFee := MiningFee(amount)

	expectedTotal := uint64(100_000_000)
	expectedDevelopment := uint64(10_000_000)
	expectedMining := uint64(90_000_000)

	if totalFee != expectedTotal {
		t.Fatalf(
			"expected total fee %d, got %d",
			expectedTotal,
			totalFee,
		)
	}

	if developmentFee != expectedDevelopment {
		t.Fatalf(
			"expected development fee %d, got %d",
			expectedDevelopment,
			developmentFee,
		)
	}

	if miningFee != expectedMining {
		t.Fatalf(
			"expected mining fee %d, got %d",
			expectedMining,
			miningFee,
		)
	}

	if developmentFee+miningFee != totalFee {
		t.Fatal("fee allocation does not equal total fee")
	}
}

func TestTransactionSigning(t *testing.T) {
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
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if !tx.Verify() {
		t.Fatal("transaction signature verification failed")
	}
}

func TestWrongWalletCannotSign(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	wrongWallet, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := NewTransaction(
		sender.Address,
		"receiver-address",
		100*100_000_000,
	)

	if err := tx.Sign(wrongWallet); err == nil {
		t.Fatal(
			"wrong wallet should not be able to sign transaction",
		)
	}
}

func TestAmountTamperingFails(t *testing.T) {
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
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if !tx.Verify() {
		t.Fatal("original transaction should be valid")
	}

	// Attacker changes 100 SUDH to 1,000 SUDH.
	tx.Amount = 1000 * 100_000_000

	if tx.Verify() {
		t.Fatal(
			"tampered transaction amount was accepted",
		)
	}
}

func TestReceiverTamperingFails(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	attacker, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := NewTransaction(
		sender.Address,
		receiver.Address,
		100*100_000_000,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	tx.To = attacker.Address

	if tx.Verify() {
		t.Fatal(
			"transaction with tampered receiver was accepted",
		)
	}
}

func TestFeeTamperingFails(t *testing.T) {
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
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	// Attacker attempts to remove the transaction fee.
	tx.Fee = 0

	if tx.Verify() {
		t.Fatal(
			"transaction with tampered fee was accepted",
		)
	}
}

func TestSignatureTamperingFails(t *testing.T) {
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
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if len(tx.Signature) == 0 {
		t.Fatal("signature was not created")
	}

	// Corrupt one byte of the signature.
	tx.Signature[0] ^= 0xff

	if tx.Verify() {
		t.Fatal(
			"transaction with corrupted signature was accepted",
		)
	}
}

func TestPublicKeyTamperingFails(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	attacker, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := NewTransaction(
		sender.Address,
		receiver.Address,
		100*100_000_000,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	tx.PublicKey = attacker.PublicKey

	if tx.Verify() {
		t.Fatal(
			"transaction with substituted public key was accepted",
		)
	}
}
