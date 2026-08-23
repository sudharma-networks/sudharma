package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/miner"
)

func TestEmptyBlockBroadcastBetweenNodes(t *testing.T) {
	chainA := blockchain.NewChain()
	chainB := blockchain.NewChain()
	stateA := blockchain.NewState()
	stateB := blockchain.NewState()

	nodeA, err := NewNode("block-node-a", "127.0.0.1:0", 0, chainA.Tip().Hash())
	if err != nil { t.Fatal(err) }
	nodeB, err := NewNode("block-node-b", "127.0.0.1:0", 0, chainB.Tip().Hash())
	if err != nil { t.Fatal(err) }
	if err := nodeA.SetChain(chainA); err != nil { t.Fatal(err) }
	if err := nodeB.SetChain(chainB); err != nil { t.Fatal(err) }
	if err := nodeA.SetState(stateA); err != nil { t.Fatal(err) }
	if err := nodeB.SetState(stateB); err != nil { t.Fatal(err) }
	if err := nodeA.Start(); err != nil { t.Fatal(err) }
	defer nodeA.Stop()
	if err := nodeB.Start(); err != nil { t.Fatal(err) }
	defer nodeB.Stop()
	if _, err := nodeB.Connect(nodeA.ListenAddress); err != nil { t.Fatal(err) }

	previous := chainB.Tip()
	blockTime := int64(60)
	difficulty := consensus.NextDifficulty(previous.Difficulty, blockTime)
	block := &blockchain.Block{
		Version: 1, Height: previous.Height + 1,
		Timestamp: previous.Timestamp + blockTime,
		PreviousHash: previous.Hash(), Difficulty: difficulty, Nonce: 0,
		MinerAddress: "test-miner-address", Transactions: nil,
	}
	block.UpdateMerkleRoot()
	result := miner.Mine(block, 0, 1_000_000)
	if !result.Found { t.Fatal("failed to mine test block") }
	if err := chainB.AddBlock(block); err != nil { t.Fatal(err) }
	nodeB.RefreshChainStatus()
	if err := nodeB.BroadcastBlock(block); err != nil { t.Fatal(err) }

	deadline := time.Now().Add(2 * time.Second)
	for chainA.Height() == 0 && time.Now().Before(deadline) { time.Sleep(10 * time.Millisecond) }
	if chainA.Height() != 1 { t.Fatalf("expected node A height 1, got %d", chainA.Height()) }
	if chainA.Tip().Hash() != block.Hash() { t.Fatal("node A tip does not match broadcast block") }
	height, _ := nodeA.AdvertisedChainStatus()
	if height != 1 { t.Fatalf("expected node A advertised height 1, got %d", height) }
}
