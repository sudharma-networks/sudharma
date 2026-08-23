package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func newMempoolTestState(
	t *testing.T,
	senderBalance uint64,
) (*State, *wallet.Wallet, *wallet.Wallet) {

	t.Helper()

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	development, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()

	state.SetDevelopmentAddress(
		development.Address,
	)

	if senderBalance > 0 {
		if err := state.Credit(
			sender.Address,
			senderBalance,
		); err != nil {
			t.Fatal(err)
		}
	}

	return state, sender, development
}

func TestMempoolAcceptsFundedTransaction(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			1000*params.CoinDecimals,
		)

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		100*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if err := ValidateMempoolTransaction(
		state,
		nil,
		tx,
	); err != nil {

		t.Fatalf(
			"valid funded transaction rejected: %v",
			err,
		)
	}
}

func TestMempoolRejectsZeroBalance(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			0,
		)

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if err := ValidateMempoolTransaction(
		state,
		nil,
		tx,
	); err == nil {

		t.Fatal(
			"zero-balance transaction was accepted",
		)
	}
}

func TestMempoolRejectsOverspending(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			50*params.CoinDecimals,
		)

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		100*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if err := ValidateMempoolTransaction(
		state,
		nil,
		tx,
	); err == nil {

		t.Fatal(
			"overspending transaction was accepted",
		)
	}
}

func TestMempoolAcceptsNextPendingNonce(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			1000*params.CoinDecimals,
		)

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx1 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx1.Sign(sender); err != nil {
		t.Fatal(err)
	}

	tx2 := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		20*params.CoinDecimals,
		2,
	)

	if err := tx2.Sign(sender); err != nil {
		t.Fatal(err)
	}

	pending :=
		[]*transactions.Transaction{
			tx1,
		}

	if err := ValidateMempoolTransaction(
		state,
		pending,
		tx2,
	); err != nil {

		t.Fatalf(
			"valid nonce-2 transaction rejected: %v",
			err,
		)
	}
}

func TestMempoolRejectsDuplicateNonce(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			1000*params.CoinDecimals,
		)

	receiver1, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver2, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx1 := transactions.NewTransaction(
		sender.Address,
		receiver1.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx1.Sign(sender); err != nil {
		t.Fatal(err)
	}

	// Different transaction ID,
	// but illegally reuses nonce 1.
	txConflict := transactions.NewTransaction(
		sender.Address,
		receiver2.Address,
		20*params.CoinDecimals,
		1,
	)

	if err := txConflict.Sign(sender); err != nil {
		t.Fatal(err)
	}

	pending :=
		[]*transactions.Transaction{
			tx1,
		}

	if err := ValidateMempoolTransaction(
		state,
		pending,
		txConflict,
	); err == nil {

		t.Fatal(
			"duplicate sender nonce was accepted",
		)
	}
}

func TestMempoolRejectsDuplicateTransaction(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			1000*params.CoinDecimals,
		)

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	pending :=
		[]*transactions.Transaction{
			tx,
		}

	if err := ValidateMempoolTransaction(
		state,
		pending,
		tx,
	); err == nil {

		t.Fatal(
			"duplicate transaction was accepted",
		)
	}
}

func TestMempoolRejectsTamperedTransaction(t *testing.T) {
	state, sender, _ :=
		newMempoolTestState(
			t,
			1000*params.CoinDecimals,
		)

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		10*params.CoinDecimals,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	// Tamper after signature creation.
	tx.Amount =
		100 * params.CoinDecimals

	if err := ValidateMempoolTransaction(
		state,
		nil,
		tx,
	); err == nil {

		t.Fatal(
			"tampered transaction was accepted",
		)
	}
}
