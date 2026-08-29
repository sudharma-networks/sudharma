package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

// TestBlockGossipTriangleNoDuplicateCommit proves that the same block
// cannot be applied multiple times when it reaches a node through
// multiple network paths.
//
// Topology:
//
//	  A
//	 / \
//	B---C
//
// A broadcasts block #1 to both B and C.
// B and C may also relay that same block to each other.
//
// Expected:
//   - all nodes finish at height 1
//   - reward is applied exactly once on every node
//   - issued supply remains exactly 50 SUDH
//   - no node advances to height 2 from duplicate delivery
func TestBlockGossipTriangleNoDuplicateCommit(t *testing.T) {

	chainA := blockchain.NewChain()
	chainB := blockchain.NewChain()
	chainC := blockchain.NewChain()

	stateA := blockchain.NewState()
	stateB := blockchain.NewState()
	stateC := blockchain.NewState()

	nodeA, err := NewNode(
		"duplicate-block-a",
		"127.0.0.1:0",
		chainA.Height(),
		chainA.Tip().Hash(),
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"duplicate-block-b",
		"127.0.0.1:0",
		chainB.Height(),
		chainB.Tip().Hash(),
	)

	if err != nil {
		t.Fatal(err)
	}

	nodeC, err := NewNode(
		"duplicate-block-c",
		"127.0.0.1:0",
		chainC.Height(),
		chainC.Tip().Hash(),
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

	if err := nodeC.SetChain(chainC); err != nil {
		t.Fatal(err)
	}

	if err := nodeA.SetState(stateA); err != nil {
		t.Fatal(err)
	}

	if err := nodeB.SetState(stateB); err != nil {
		t.Fatal(err)
	}

	if err := nodeC.SetState(stateC); err != nil {
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

	if err := nodeC.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeC.Stop()

	// Build triangle:
	//
	// A <-> B
	// B <-> C
	// C <-> A

	if _, err := nodeA.Connect(
		nodeB.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeB.Connect(
		nodeC.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeC.Connect(
		nodeA.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	// Connect() completes the outbound side after the handshake response, while
	// the remote listener stores its inbound peer immediately afterward in its
	// connection goroutine. Under the race detector that final registration can
	// still be in flight here, so wait for the intended topology before testing
	// the block-gossip behavior itself.
	peerDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(peerDeadline) {
		if nodeA.PeerCount() == 2 && nodeB.PeerCount() == 2 && nodeC.PeerCount() == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if nodeA.PeerCount() != 2 {
		t.Fatalf(
			"expected node A peer count 2, got %d",
			nodeA.PeerCount(),
		)
	}

	if nodeB.PeerCount() != 2 {
		t.Fatalf(
			"expected node B peer count 2, got %d",
			nodeB.PeerCount(),
		)
	}

	if nodeC.PeerCount() != 2 {
		t.Fatalf(
			"expected node C peer count 2, got %d",
			nodeC.PeerCount(),
		)
	}

	minerWallet, err := wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	result, reward, err :=
		miner.MineNextBlock(
			chainA,
			stateA,
			nodeA.Mempool(),
			minerWallet.Address,
			1_000_000,
		)

	if err != nil {
		t.Fatal(err)
	}

	if !result.Found {
		t.Fatal(
			"failed to mine block",
		)
	}

	nodeA.RefreshChainStatus()

	if err := nodeA.BroadcastBlock(
		result.Block,
	); err != nil {
		t.Fatal(err)
	}

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for time.Now().Before(deadline) {

		if chainB.Height() == 1 &&
			chainC.Height() == 1 {

			break
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if chainB.Height() != 1 {
		t.Fatalf(
			"expected node B height 1, got %d",
			chainB.Height(),
		)
	}

	if chainC.Height() != 1 {
		t.Fatalf(
			"expected node C height 1, got %d",
			chainC.Height(),
		)
	}

	expectedHash :=
		result.Block.Hash()

	if chainB.Tip().Hash() != expectedHash {
		t.Fatal(
			"node B has wrong block tip",
		)
	}

	if chainC.Tip().Hash() != expectedHash {
		t.Fatal(
			"node C has wrong block tip",
		)
	}

	expectedReward :=
		uint64(50) *
			params.CoinDecimals

	if reward != expectedReward {
		t.Fatalf(
			"expected reward %d, got %d",
			expectedReward,
			reward,
		)
	}

	// Give duplicate relay paths enough time to arrive.
	time.Sleep(
		500 * time.Millisecond,
	)

	// Duplicate block delivery must NOT create another block.
	if chainA.Height() != 1 {
		t.Fatalf(
			"duplicate delivery changed node A height to %d",
			chainA.Height(),
		)
	}

	if chainB.Height() != 1 {
		t.Fatalf(
			"duplicate delivery changed node B height to %d",
			chainB.Height(),
		)
	}

	if chainC.Height() != 1 {
		t.Fatalf(
			"duplicate delivery changed node C height to %d",
			chainC.Height(),
		)
	}

	// Supply must be created only once.
	if stateA.IssuedSupply() != expectedReward {
		t.Fatalf(
			"node A supply duplicated: expected %d, got %d",
			expectedReward,
			stateA.IssuedSupply(),
		)
	}

	if stateB.IssuedSupply() != expectedReward {
		t.Fatalf(
			"node B supply duplicated: expected %d, got %d",
			expectedReward,
			stateB.IssuedSupply(),
		)
	}

	if stateC.IssuedSupply() != expectedReward {
		t.Fatalf(
			"node C supply duplicated: expected %d, got %d",
			expectedReward,
			stateC.IssuedSupply(),
		)
	}

	// Miner reward must also appear only once.
	if stateA.Balance(
		minerWallet.Address,
	) != expectedReward {

		t.Fatalf(
			"node A miner reward duplicated",
		)
	}

	if stateB.Balance(
		minerWallet.Address,
	) != expectedReward {

		t.Fatalf(
			"node B miner reward duplicated",
		)
	}

	if stateC.Balance(
		minerWallet.Address,
	) != expectedReward {

		t.Fatalf(
			"node C miner reward duplicated",
		)
	}
}
