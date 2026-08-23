package p2p

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMempoolSaveAndLoad(t *testing.T) {
	sender, err :=
		wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	receiver, err :=
		wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	state :=
		blockchain.NewState()

	initialBalance :=
		uint64(100) *
			params.CoinDecimals

	if err :=
		state.Credit(
			sender.Address,
			initialBalance,
		); err != nil {

		t.Fatal(err)
	}

	nodeA, err :=
		NewNode(
			"mempool-save-a",
			"127.0.0.1:0",
			0,
			blockchain.NewGenesisBlock().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		nodeA.SetState(
			state,
		); err != nil {

		t.Fatal(err)
	}

	amount :=
		uint64(10) *
			params.CoinDecimals

	tx :=
		transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			amount,
			1,
		)

	if err :=
		tx.Sign(
			sender,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		blockchain.ValidateMempoolTransaction(
			state,
			nodeA.Mempool().AllTransactions(),
			tx,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		nodeA.Mempool().AddTransaction(
			tx,
		); err != nil {

		t.Fatal(err)
	}

	tempDirectory :=
		t.TempDir()

	path :=
		filepath.Join(
			tempDirectory,
			"sudharma-mempool.json",
		)

	if err :=
		nodeA.SaveMempoolToFile(
			path,
		); err != nil {

		t.Fatal(err)
	}

	nodeB, err :=
		NewNode(
			"mempool-save-b",
			"127.0.0.1:0",
			0,
			blockchain.NewGenesisBlock().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		nodeB.SetState(
			state.Clone(),
		); err != nil {

		t.Fatal(err)
	}

	loaded,
		rejected,
		err :=
		nodeB.LoadMempoolFromFile(
			path,
		)

	if err != nil {
		t.Fatal(err)
	}

	if loaded != 1 {
		t.Fatalf(
			"expected 1 loaded transaction, got %d",
			loaded,
		)
	}

	if rejected != 0 {
		t.Fatalf(
			"expected 0 rejected transactions, got %d",
			rejected,
		)
	}

	if nodeB.MempoolCount() != 1 {
		t.Fatalf(
			"expected mempool count 1, got %d",
			nodeB.MempoolCount(),
		)
	}

	restored, ok :=
		nodeB.MempoolTransaction(
			tx.ID,
		)

	if !ok {
		t.Fatal(
			"saved transaction was not restored",
		)
	}

	if restored.ID != tx.ID {
		t.Fatal(
			"restored wrong transaction",
		)
	}
}

func TestConfirmedStoredTransactionRejected(t *testing.T) {
	sender, err :=
		wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	receiver, err :=
		wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	state :=
		blockchain.NewState()

	initialBalance :=
		uint64(100) *
			params.CoinDecimals

	if err :=
		state.Credit(
			sender.Address,
			initialBalance,
		); err != nil {

		t.Fatal(err)
	}

	tx :=
		transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			uint64(10)*params.CoinDecimals,
			1,
		)

	if err :=
		tx.Sign(
			sender,
		); err != nil {

		t.Fatal(err)
	}

	saveNode, err :=
		NewNode(
			"confirmed-save",
			"127.0.0.1:0",
			0,
			blockchain.NewGenesisBlock().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		saveNode.SetState(
			state.Clone(),
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		saveNode.Mempool().AddTransaction(
			tx,
		); err != nil {

		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"sudharma-mempool.json",
		)

	if err :=
		saveNode.SaveMempoolToFile(
			path,
		); err != nil {

		t.Fatal(err)
	}

	confirmedState :=
		state.Clone()

	if _, err :=
		blockchain.ApplyTransaction(
			confirmedState,
			tx,
		); err != nil {

		t.Fatal(err)
	}

	loadNode, err :=
		NewNode(
			"confirmed-load",
			"127.0.0.1:0",
			0,
			blockchain.NewGenesisBlock().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		loadNode.SetState(
			confirmedState,
		); err != nil {

		t.Fatal(err)
	}

	loaded,
		rejected,
		err :=
		loadNode.LoadMempoolFromFile(
			path,
		)

	if err != nil {
		t.Fatal(err)
	}

	if loaded != 0 {
		t.Fatalf(
			"expected 0 loaded transactions, got %d",
			loaded,
		)
	}

	if rejected != 1 {
		t.Fatalf(
			"expected 1 rejected transaction, got %d",
			rejected,
		)
	}

	if loadNode.MempoolCount() != 0 {
		t.Fatal(
			"confirmed transaction was incorrectly restored",
		)
	}
}

func TestCorruptedMempoolFileRejected(t *testing.T) {
	state :=
		blockchain.NewState()

	node, err :=
		NewNode(
			"corrupted-mempool",
			"127.0.0.1:0",
			0,
			blockchain.NewGenesisBlock().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		node.SetState(
			state,
		); err != nil {

		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"sudharma-mempool.json",
		)

	if err :=
		os.WriteFile(
			path,
			[]byte(
				"{this is not valid json",
			),
			0o600,
		); err != nil {

		t.Fatal(err)
	}

	if _, _, err :=
		node.LoadMempoolFromFile(
			path,
		); err == nil {

		t.Fatal(
			"corrupted mempool file was accepted",
		)
	}
}

func TestMissingMempoolFileCreatesEmptyPool(t *testing.T) {
	state :=
		blockchain.NewState()

	node, err :=
		NewNode(
			"missing-mempool",
			"127.0.0.1:0",
			0,
			blockchain.NewGenesisBlock().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		node.SetState(
			state,
		); err != nil {

		t.Fatal(err)
	}

	path :=
		filepath.Join(
			t.TempDir(),
			"does-not-exist.json",
		)

	loaded,
		rejected,
		err :=
		node.LoadMempoolFromFile(
			path,
		)

	if err != nil {
		t.Fatal(err)
	}

	if loaded != 0 ||
		rejected != 0 {

		t.Fatalf(
			"unexpected counts: loaded=%d rejected=%d",
			loaded,
			rejected,
		)
	}

	if node.MempoolCount() != 0 {
		t.Fatal(
			"missing mempool file produced transactions",
		)
	}
}
