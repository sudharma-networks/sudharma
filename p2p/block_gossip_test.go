package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestBlockGossipThreeNodes(t *testing.T) {
	chainA := blockchain.NewChain()
	chainB := blockchain.NewChain()
	chainC := blockchain.NewChain()

	stateA := blockchain.NewState()
	stateB := blockchain.NewState()
	stateC := blockchain.NewState()

	nodeA, err := NewNode(
		"block-gossip-a",
		"127.0.0.1:0",
		chainA.Height(),
		chainA.Tip().Hash(),
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"block-gossip-b",
		"127.0.0.1:0",
		chainB.Height(),
		chainB.Tip().Hash(),
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeC, err := NewNode(
		"block-gossip-c",
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

	if _, err := nodeA.Connect(nodeB.ListenAddress); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeB.Connect(nodeC.ListenAddress); err != nil {
		t.Fatal(err)
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
		t.Fatal("node A failed to mine block")
	}

	nodeA.RefreshChainStatus()

	if err := nodeA.BroadcastBlock(result.Block); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if chainB.Height() == 1 && chainC.Height() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if chainB.Height() != 1 {
		t.Fatalf("expected node B height 1, got %d", chainB.Height())
	}

	if chainC.Height() != 1 {
		t.Fatalf("expected node C height 1, got %d", chainC.Height())
	}

	expectedHash := result.Block.Hash()

	if chainB.Tip().Hash() != expectedHash {
		t.Fatal("node B tip does not match mined block")
	}

	if chainC.Tip().Hash() != expectedHash {
		t.Fatal("node C tip does not match relayed block")
	}

	expectedReward := uint64(50) * params.CoinDecimals

	if reward != expectedReward {
		t.Fatalf(
			"wrong mined reward: expected %d, got %d",
			expectedReward,
			reward,
		)
	}

	if stateB.Balance(minerWallet.Address) != expectedReward {
		t.Fatalf(
			"node B miner balance mismatch: expected %d, got %d",
			expectedReward,
			stateB.Balance(minerWallet.Address),
		)
	}

	if stateC.Balance(minerWallet.Address) != expectedReward {
		t.Fatalf(
			"node C miner balance mismatch: expected %d, got %d",
			expectedReward,
			stateC.Balance(minerWallet.Address),
		)
	}

	time.Sleep(200 * time.Millisecond)

	if chainB.Height() != 1 {
		t.Fatalf("duplicate gossip changed node B height to %d", chainB.Height())
	}

	if chainC.Height() != 1 {
		t.Fatalf("duplicate gossip changed node C height to %d", chainC.Height())
	}
}
