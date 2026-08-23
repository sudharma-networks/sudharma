package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestBlockRollbackOnInvalidTransaction(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver1, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver2, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	development, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	minerWallet, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()
	state.SetDevelopmentAddress(development.Address)

	initialBalance := uint64(150 * 100_000_000)
	state.Credit(sender.Address, initialBalance)

	// First transaction is valid.
	tx1 := transactions.NewTransaction(
		sender.Address,
		receiver1.Address,
		100*100_000_000,
		1,
	)

	if err := tx1.Sign(sender); err != nil {
		t.Fatal(err)
	}

	// Second transaction has the correct next nonce,
	// but there will not be enough balance after tx1.
	tx2 := transactions.NewTransaction(
		sender.Address,
		receiver2.Address,
		100*100_000_000,
		2,
	)

	if err := tx2.Sign(sender); err != nil {
		t.Fatal(err)
	}

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    1786924860,
		PreviousHash: NewGenesisBlock().Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: []*transactions.Transaction{
			tx1,
			tx2,
		},
	}

	block.UpdateMerkleRoot()

	_, err = ProcessBlock(
		state,
		block,
		minerWallet.Address,
	)

	if err == nil {
		t.Fatal("expected block processing to fail")
	}

	// Entire block must roll back.
	if state.Balance(sender.Address) != initialBalance {
		t.Fatalf(
			"sender balance changed after rejected block: expected %d, got %d",
			initialBalance,
			state.Balance(sender.Address),
		)
	}

	if state.Balance(receiver1.Address) != 0 {
		t.Fatal("receiver1 received funds from rejected block")
	}

	if state.Balance(receiver2.Address) != 0 {
		t.Fatal("receiver2 received funds from rejected block")
	}

	if state.Balance(development.Address) != 0 {
		t.Fatal("development treasury changed after rejected block")
	}

	if state.Balance(minerWallet.Address) != 0 {
		t.Fatal("miner received reward from rejected block")
	}

	if state.AccountNonce(sender.Address) != 0 {
		t.Fatal("sender nonce changed after rejected block")
	}

	if state.IsTransactionProcessed(tx1.ID) {
		t.Fatal("tx1 marked processed after rejected block")
	}

	if state.IsTransactionProcessed(tx2.ID) {
		t.Fatal("tx2 marked processed after rejected block")
	}
}
