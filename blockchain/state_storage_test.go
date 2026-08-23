package blockchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestSaveAndLoadState(t *testing.T) {
	state := NewState()

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	initialBalance :=
		uint64(100) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialBalance,
	); err != nil {
		t.Fatal(err)
	}

	amount :=
		uint64(10) *
			params.CoinDecimals

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	if _, err :=
		ApplyTransaction(
			state,
			tx,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		state.MintSupply(
			params.InitialBlockReward,
		); err != nil {

		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"sudharma-state.json",
		)

	if err := state.SaveToFile(
		path,
	); err != nil {

		t.Fatalf(
			"failed to save state: %v",
			err,
		)
	}

	loaded, err :=
		LoadStateFromFile(
			path,
		)

	if err != nil {
		t.Fatalf(
			"failed to load state: %v",
			err,
		)
	}

	if loaded.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatalf(
			"wrong treasury after reload: expected %s, got %s",
			params.DevelopmentTreasuryAddress,
			loaded.DevelopmentAddress(),
		)
	}

	if loaded.Balance(
		sender.Address,
	) != state.Balance(
		sender.Address,
	) {

		t.Fatalf(
			"sender balance changed after reload",
		)
	}

	if loaded.Balance(
		receiver.Address,
	) != state.Balance(
		receiver.Address,
	) {

		t.Fatalf(
			"receiver balance changed after reload",
		)
	}

	if loaded.Balance(
		params.DevelopmentTreasuryAddress,
	) != state.Balance(
		params.DevelopmentTreasuryAddress,
	) {

		t.Fatalf(
			"treasury balance changed after reload",
		)
	}

	if loaded.AccountNonce(
		sender.Address,
	) != 1 {

		t.Fatalf(
			"expected sender nonce 1 after reload, got %d",
			loaded.AccountNonce(sender.Address),
		)
	}

	if !loaded.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"processed transaction was lost after reload",
		)
	}

	if loaded.IssuedSupply() !=
		params.InitialBlockReward {

		t.Fatalf(
			"wrong issued supply after reload: expected %d, got %d",
			params.InitialBlockReward,
			loaded.IssuedSupply(),
		)
	}
}

func TestCorruptedStateFileRejected(t *testing.T) {
	path :=
		filepath.Join(
			t.TempDir(),
			"bad-state.json",
		)

	badData :=
		[]byte(
			`{"version":1,"development_address":"attacker-address"}`,
		)

	if err := os.WriteFile(
		path,
		badData,
		0600,
	); err != nil {

		t.Fatal(err)
	}

	if _, err :=
		LoadStateFromFile(
			path,
		); err == nil {

		t.Fatal(
			"corrupted or invalid treasury state was accepted",
		)
	}
}

func TestStateSaveReloadTwice(t *testing.T) {
	state := NewState()

	address :=
		"test-persistent-address"

	if err := state.Credit(
		address,
		123456789,
	); err != nil {
		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"state.json",
		)

	if err := state.SaveToFile(
		path,
	); err != nil {
		t.Fatal(err)
	}

	loaded, err :=
		LoadStateFromFile(
			path,
		)

	if err != nil {
		t.Fatal(err)
	}

	if err := loaded.SaveToFile(
		path,
	); err != nil {

		t.Fatalf(
			"failed to replace existing state file: %v",
			err,
		)
	}

	loadedAgain, err :=
		LoadStateFromFile(
			path,
		)

	if err != nil {
		t.Fatal(err)
	}

	if loadedAgain.Balance(
		address,
	) != 123456789 {

		t.Fatalf(
			"wrong balance after second reload: %d",
			loadedAgain.Balance(address),
		)
	}
}
