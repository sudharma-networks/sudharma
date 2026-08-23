package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/miner"
)

func addSyncTestBlock(
	t *testing.T,
	chain *blockchain.Chain,
	minerAddress string,
) {
	t.Helper()

	previous := chain.Tip()

	if previous == nil {
		t.Fatal(
			"chain tip is nil",
		)
	}

	blockTime := int64(60)

	block := &blockchain.Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + blockTime,
		PreviousHash: previous.Hash(),
		Difficulty: consensus.NextDifficulty(
			previous.Difficulty,
			blockTime,
		),
		MinerAddress: minerAddress,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	result := miner.Mine(
		block,
		0,
		1_000_000,
	)

	if !result.Found {
		t.Fatal(
			"failed to mine sync test block",
		)
	}

	if err := chain.AddBlock(
		block,
	); err != nil {
		t.Fatal(err)
	}
}

func TestRequestBlocksFromPeer(t *testing.T) {
	chainA := blockchain.NewChain()
	chainB := blockchain.NewChain()

	addSyncTestBlock(
		t,
		chainA,
		"sync-test-miner",
	)

	addSyncTestBlock(
		t,
		chainA,
		"sync-test-miner",
	)

	addSyncTestBlock(
		t,
		chainA,
		"sync-test-miner",
	)

	if chainA.Height() != 3 {
		t.Fatalf(
			"expected node A height 3, got %d",
			chainA.Height(),
		)
	}

	if chainB.Height() != 0 {
		t.Fatalf(
			"expected node B height 0, got %d",
			chainB.Height(),
		)
	}

	nodeA, err := NewNode(
		"sync-node-a",
		"127.0.0.1:0",
		chainA.Height(),
		chainA.Tip().Hash(),
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"sync-node-b",
		"127.0.0.1:0",
		chainB.Height(),
		chainB.Tip().Hash(),
	)

	if err != nil {
		t.Fatal(err)
	}

	if err := nodeA.SetChain(
		chainA,
	); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.SetChain(
		chainB,
	); err != nil {
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

	peer, err := nodeB.Connect(
		nodeA.ListenAddress,
	)

	if err != nil {
		t.Fatal(err)
	}

	blocks, err := nodeB.RequestBlocks(
		peer.NodeID,
		1,
		MaxBlocksPerMessage,
		2*time.Second,
	)

	if err != nil {
		t.Fatalf(
			"block request failed: %v",
			err,
		)
	}

	if len(blocks) != 3 {
		t.Fatalf(
			"expected 3 blocks, got %d",
			len(blocks),
		)
	}

	for i, block := range blocks {
		expectedHeight := uint64(i + 1)

		if block.Height != expectedHeight {
			t.Fatalf(
				"expected block height %d, got %d",
				expectedHeight,
				block.Height,
			)
		}
	}

	if blocks[2].Hash() !=
		chainA.Tip().Hash() {

		t.Fatal(
			"last downloaded block does not match node A tip",
		)
	}

	// Step 47B is transport only.
	// Node B's chain must still be untouched.
	if chainB.Height() != 0 {
		t.Fatalf(
			"node B chain changed during transport test; height %d",
			chainB.Height(),
		)
	}
}

func TestBlocksFromHeightHonorsLimit(t *testing.T) {
	chain := blockchain.NewChain()

	for i := 0; i < 5; i++ {
		addSyncTestBlock(
			t,
			chain,
			"sync-limit-miner",
		)
	}

	node, err := NewNode(
		"sync-limit-node",
		"127.0.0.1:0",
		chain.Height(),
		chain.Tip().Hash(),
	)

	if err != nil {
		t.Fatal(err)
	}

	if err := node.SetChain(
		chain,
	); err != nil {
		t.Fatal(err)
	}

	blocks, err := node.blocksFromHeight(
		2,
		2,
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 2 {
		t.Fatalf(
			"expected 2 blocks, got %d",
			len(blocks),
		)
	}

	if blocks[0].Height != 2 ||
		blocks[1].Height != 3 {

		t.Fatal(
			"wrong block range returned",
		)
	}
}
