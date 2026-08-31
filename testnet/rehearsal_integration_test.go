package testnet

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/wallet"
)

// TestPublicTestnetRehearsal exercises the lifecycle we expect before public
// deployment: a seed advances the chain, a fresh peer joins and synchronizes,
// the peer persists its chain/state, and the persisted data survives restart.
func TestPublicTestnetRehearsal(t *testing.T) {
	seedChain := blockchain.NewChain()
	seedState := blockchain.NewState()
	seedPool := mempool.NewMempool()

	minerWallet, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		result, _, err := miner.MineNextBlock(seedChain, seedState, seedPool, minerWallet.Address, 1_000_000)
		if err != nil {
			t.Fatalf("mine rehearsal block %d: %v", i+1, err)
		}
		if !result.Found {
			t.Fatalf("rehearsal block %d was not found", i+1)
		}
	}

	joinChain := blockchain.NewChain()
	joinState := blockchain.NewState()

	seed, err := p2p.NewNode("testnet-rehearsal-seed", "127.0.0.1:0", seedChain.Height(), seedChain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	joiner, err := p2p.NewNode("testnet-rehearsal-joiner", "127.0.0.1:0", joinChain.Height(), joinChain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}

	if err := seed.SetChain(seedChain); err != nil {
		t.Fatal(err)
	}
	if err := seed.SetState(seedState); err != nil {
		t.Fatal(err)
	}
	if err := joiner.SetChain(joinChain); err != nil {
		t.Fatal(err)
	}
	if err := joiner.SetState(joinState); err != nil {
		t.Fatal(err)
	}

	if err := seed.Start(); err != nil {
		t.Fatal(err)
	}
	defer seed.Stop()
	if err := joiner.Start(); err != nil {
		t.Fatal(err)
	}

	peer, err := joiner.Connect(seed.ListenAddress)
	if err != nil {
		_ = joiner.Stop()
		t.Fatal(err)
	}
	if err := joiner.SyncFromPeer(peer.NodeID, 3*time.Second); err != nil {
		_ = joiner.Stop()
		t.Fatalf("testnet peer synchronization failed: %v", err)
	}

	if joinChain.Height() != seedChain.Height() || joinChain.Tip().Hash() != seedChain.Tip().Hash() {
		_ = joiner.Stop()
		t.Fatalf("testnet nodes diverged after sync: seed=%d/%s joiner=%d/%s", seedChain.Height(), seedChain.Tip().Hash(), joinChain.Height(), joinChain.Tip().Hash())
	}
	if joinState.IssuedSupply() != seedState.IssuedSupply() {
		_ = joiner.Stop()
		t.Fatalf("state supply mismatch after sync: seed=%d joiner=%d", seedState.IssuedSupply(), joinState.IssuedSupply())
	}

	dataDir := t.TempDir()
	chainPath := filepath.Join(dataDir, "sudharma-chain.json")
	statePath := filepath.Join(dataDir, "sudharma-state.json")
	if err := joinChain.SaveToFile(chainPath); err != nil {
		t.Fatal(err)
	}
	if err := joinState.SaveToFile(statePath); err != nil {
		t.Fatal(err)
	}
	if err := joiner.Stop(); err != nil {
		t.Fatal(err)
	}

	restartedChain, err := blockchain.LoadChainFromFile(chainPath)
	if err != nil {
		t.Fatal(err)
	}
	restartedState, err := blockchain.LoadStateFromFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if restartedChain.Height() != seedChain.Height() || restartedChain.Tip().Hash() != seedChain.Tip().Hash() {
		t.Fatalf("persisted chain did not survive restart")
	}
	if restartedState.IssuedSupply() != seedState.IssuedSupply() {
		t.Fatalf("persisted state did not survive restart")
	}

	restartedNode, err := p2p.NewNode("testnet-rehearsal-restarted", "127.0.0.1:0", restartedChain.Height(), restartedChain.Tip().Hash())
	if err != nil {
		t.Fatal(err)
	}
	if err := restartedNode.SetChain(restartedChain); err != nil {
		t.Fatal(err)
	}
	if err := restartedNode.SetState(restartedState); err != nil {
		t.Fatal(err)
	}
	if err := restartedNode.Start(); err != nil {
		t.Fatal(err)
	}
	if err := restartedNode.Stop(); err != nil {
		t.Fatal(err)
	}
}
