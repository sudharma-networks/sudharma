package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestOrphanedTransactionReturnsToMempool(t *testing.T) {
	// -------------------------------------------------
	// Shared wallet.
	//
	// This wallet receives mining rewards on BOTH forks,
	// so the orphaned transaction remains spendable after
	// reorganization.
	// -------------------------------------------------

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

	// =================================================
	// CURRENT CHAIN
	// =================================================

	current :=
		blockchain.NewChain()

	currentState :=
		blockchain.NewState()

	node, err :=
		NewNode(
			"reorg-mempool-node",
			"127.0.0.1:0",
			current.Height(),
			current.Tip().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		node.SetChain(
			current,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		node.SetState(
			currentState,
		); err != nil {

		t.Fatal(err)
	}

	// Mine current block #1.
	result, _, err :=
		miner.MineNextBlock(
			current,
			currentState,
			node.Mempool(),
			sender.Address,
			1_000_000,
		)

	if err != nil {
		t.Fatal(err)
	}

	if !result.Found {
		t.Fatal(
			"failed to mine current block 1",
		)
	}

	// Sender now owns 50 SUDH.
	amount :=
		uint64(10) *
			params.CoinDecimals

	nonce, err :=
		currentState.ExpectedNonce(
			sender.Address,
		)

	if err != nil {
		t.Fatal(err)
	}

	tx :=
		transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			amount,
			nonce,
		)

	if err :=
		tx.Sign(
			sender,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		blockchain.ValidateMempoolTransaction(
			currentState,
			node.Mempool().AllTransactions(),
			tx,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		node.Mempool().AddTransaction(
			tx,
		); err != nil {

		t.Fatal(err)
	}

	// Mine current block #2 containing tx.
	result, _, err =
		miner.MineNextBlock(
			current,
			currentState,
			node.Mempool(),
			sender.Address,
			1_000_000,
		)

	if err != nil {
		t.Fatal(err)
	}

	if !result.Found {
		t.Fatal(
			"failed to mine current block 2",
		)
	}

	if !currentState.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"transaction was not confirmed on old fork",
		)
	}

	if node.MempoolCount() != 0 {
		t.Fatal(
			"old fork mempool should be empty",
		)
	}

	// =================================================
	// CANDIDATE FORK
	// =================================================

	candidate :=
		blockchain.NewChain()

	candidateState :=
		blockchain.NewState()

	candidatePool :=
		mempool.NewMempool()

	// Mine THREE empty blocks to the same sender.
	//
	// Candidate therefore has more cumulative work,
	// while tx is not included in it.
	for i := 0; i < 3; i++ {

		result, _, err :=
			miner.MineNextBlock(
				candidate,
				candidateState,
				candidatePool,
				sender.Address,
				1_000_000,
			)

		if err != nil {
			t.Fatal(err)
		}

		if !result.Found {
			t.Fatalf(
				"failed to mine candidate block %d",
				i+1,
			)
		}
	}

	if candidate.TotalWork().Cmp(
		current.TotalWork(),
	) <= 0 {

		t.Fatal(
			"candidate does not have more cumulative work",
		)
	}

	// =================================================
	// REORGANIZE
	// =================================================

	adopted,
		recovered,
		err :=
		node.ReorganizeWithMempoolRecovery(
			candidate,
		)

	if err != nil {
		t.Fatal(err)
	}

	if !adopted {
		t.Fatal(
			"better candidate was not adopted",
		)
	}

	if recovered != 1 {
		t.Fatalf(
			"expected 1 recovered transaction, got %d",
			recovered,
		)
	}

	if current.Height() != 3 {
		t.Fatalf(
			"expected height 3 after reorg, got %d",
			current.Height(),
		)
	}

	// Transaction must no longer be confirmed.
	if currentState.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"orphaned transaction remained confirmed",
		)
	}

	// Receiver's abandoned-fork balance disappears.
	if currentState.Balance(
		receiver.Address,
	) != 0 {

		t.Fatalf(
			"receiver retained orphaned balance: %d",
			currentState.Balance(receiver.Address),
		)
	}

	// Transaction should now be pending again.
	if node.MempoolCount() != 1 {
		t.Fatalf(
			"expected mempool count 1, got %d",
			node.MempoolCount(),
		)
	}

	recoveredTx, ok :=
		node.MempoolTransaction(
			tx.ID,
		)

	if !ok {
		t.Fatal(
			"orphaned transaction was not returned to mempool",
		)
	}

	if recoveredTx.ID != tx.ID {
		t.Fatal(
			"wrong transaction recovered",
		)
	}
}
