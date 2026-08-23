package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestNodeSynchronizesMissingBlocks(t *testing.T) {
	// ------------------------------------------------
	// Node A starts with a blockchain and state.
	// ------------------------------------------------

	chainA :=
		blockchain.NewChain()

	stateA :=
		blockchain.NewState()

	poolA :=
		mempool.NewMempool()

	minerWallet, err :=
		wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	// ------------------------------------------------
	// Mine three real empty blocks on Node A.
	//
	// MineNextBlock processes the subsidy and updates
	// both chain and state exactly as a real node does.
	// ------------------------------------------------

	for i := 0; i < 3; i++ {
		result, _, err :=
			miner.MineNextBlock(
				chainA,
				stateA,
				poolA,
				minerWallet.Address,
				1_000_000,
			)

		if err != nil {
			t.Fatalf(
				"failed mining source block %d: %v",
				i+1,
				err,
			)
		}

		if !result.Found {
			t.Fatalf(
				"source block %d was not mined",
				i+1,
			)
		}
	}

	if chainA.Height() != 3 {
		t.Fatalf(
			"expected source height 3, got %d",
			chainA.Height(),
		)
	}

	// ------------------------------------------------
	// Node B begins at genesis only.
	// ------------------------------------------------

	chainB :=
		blockchain.NewChain()

	stateB :=
		blockchain.NewState()

	if chainB.Height() != 0 {
		t.Fatalf(
			"expected destination height 0, got %d",
			chainB.Height(),
		)
	}

	// ------------------------------------------------
	// Create both P2P nodes.
	// ------------------------------------------------

	nodeA, err :=
		NewNode(
			"full-sync-node-a",
			"127.0.0.1:0",
			chainA.Height(),
			chainA.Tip().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err :=
		NewNode(
			"full-sync-node-b",
			"127.0.0.1:0",
			chainB.Height(),
			chainB.Tip().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		nodeA.SetChain(
			chainA,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		nodeB.SetChain(
			chainB,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		nodeA.SetState(
			stateA,
		); err != nil {

		t.Fatal(err)
	}

	if err :=
		nodeB.SetState(
			stateB,
		); err != nil {

		t.Fatal(err)
	}

	// ------------------------------------------------
	// Start nodes.
	// ------------------------------------------------

	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}

	defer nodeA.Stop()

	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}

	defer nodeB.Stop()

	// ------------------------------------------------
	// Node B connects to Node A.
	// ------------------------------------------------

	peer, err :=
		nodeB.Connect(
			nodeA.ListenAddress,
		)

	if err != nil {
		t.Fatal(err)
	}

	if peer.Height != 3 {
		t.Fatalf(
			"expected remote height 3, got %d",
			peer.Height,
		)
	}

	// ------------------------------------------------
	// Full synchronization.
	// ------------------------------------------------

	if err :=
		nodeB.SyncFromPeer(
			peer.NodeID,
			2*time.Second,
		); err != nil {

		t.Fatalf(
			"chain synchronization failed: %v",
			err,
		)
	}

	// ------------------------------------------------
	// Compare chain height and tip.
	// ------------------------------------------------

	if chainB.Height() !=
		chainA.Height() {

		t.Fatalf(
			"height mismatch: node A %d, node B %d",
			chainA.Height(),
			chainB.Height(),
		)
	}

	if chainB.Tip().Hash() !=
		chainA.Tip().Hash() {

		t.Fatal(
			"node B tip does not match node A tip",
		)
	}

	if nodeB.Height != 3 {
		t.Fatalf(
			"expected node B advertised height 3, got %d",
			nodeB.Height,
		)
	}

	// ------------------------------------------------
	// Compare every downloaded block.
	// ------------------------------------------------

	for height := uint64(0); height <= chainA.Height(); height++ {

		blockA, ok :=
			chainA.BlockByHeight(
				height,
			)

		if !ok {
			t.Fatalf(
				"node A missing block %d",
				height,
			)
		}

		blockB, ok :=
			chainB.BlockByHeight(
				height,
			)

		if !ok {
			t.Fatalf(
				"node B missing block %d",
				height,
			)
		}

		if blockA.Hash() !=
			blockB.Hash() {

			t.Fatalf(
				"block %d hash mismatch",
				height,
			)
		}
	}

	// ------------------------------------------------
	// Compare resulting blockchain state.
	// ------------------------------------------------

	if stateA.Balance(
		minerWallet.Address,
	) != stateB.Balance(
		minerWallet.Address,
	) {

		t.Fatalf(
			"miner balance mismatch: node A %d, node B %d",
			stateA.Balance(minerWallet.Address),
			stateB.Balance(minerWallet.Address),
		)
	}

	if stateA.Balance(
		params.DevelopmentTreasuryAddress,
	) != stateB.Balance(
		params.DevelopmentTreasuryAddress,
	) {

		t.Fatalf(
			"development treasury balance mismatch: node A %d, node B %d",
			stateA.Balance(
				params.DevelopmentTreasuryAddress,
			),
			stateB.Balance(
				params.DevelopmentTreasuryAddress,
			),
		)
	}

	if stateA.IssuedSupply() !=
		stateB.IssuedSupply() {

		t.Fatalf(
			"issued supply mismatch: node A %d, node B %d",
			stateA.IssuedSupply(),
			stateB.IssuedSupply(),
		)
	}

	// Three 50-SUDH subsidies should have been issued.
	expectedSupply :=
		uint64(3) *
			params.InitialBlockReward

	if stateB.IssuedSupply() !=
		expectedSupply {

		t.Fatalf(
			"expected issued supply %d, got %d",
			expectedSupply,
			stateB.IssuedSupply(),
		)
	}
}

func TestSyncAlreadyUpToDate(t *testing.T) {
	chainA :=
		blockchain.NewChain()

	chainB :=
		blockchain.NewChain()

	stateA :=
		blockchain.NewState()

	stateB :=
		blockchain.NewState()

	nodeA, err :=
		NewNode(
			"up-to-date-a",
			"127.0.0.1:0",
			chainA.Height(),
			chainA.Tip().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err :=
		NewNode(
			"up-to-date-b",
			"127.0.0.1:0",
			chainB.Height(),
			chainB.Tip().Hash(),
		)

	if err != nil {
		t.Fatal(err)
	}

	if err := nodeA.SetChain(chainA); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.SetChain(chainB); err != nil {
		t.Fatal(err)
	}

	if err := nodeA.SetState(stateA); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.SetState(stateB); err != nil {
		t.Fatal(err)
	}

	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}

	defer nodeA.Stop()

	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}

	defer nodeB.Stop()

	peer, err :=
		nodeB.Connect(
			nodeA.ListenAddress,
		)

	if err != nil {
		t.Fatal(err)
	}

	if err :=
		nodeB.SyncFromPeer(
			peer.NodeID,
			2*time.Second,
		); err != nil {

		t.Fatalf(
			"up-to-date sync failed: %v",
			err,
		)
	}

	if chainB.Height() != 0 {
		t.Fatalf(
			"up-to-date chain changed height to %d",
			chainB.Height(),
		)
	}
}
