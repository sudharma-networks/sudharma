package rpc

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestPolicyMiningWorkServiceEnforcesActivationHeight(t *testing.T) {
	service := NewMiningWorkServiceWithPolicy(
		func(*blockchain.Block, uint64) bool { return true },
		blockchain.PoWPolicy{GPUV1ActivationHeight: 100},
	)

	if _, err := service.Issue(testMiningBlock(99, "miner-a")); err == nil {
		t.Fatal("issued Version 2 external work before activation")
	}
	if _, err := service.Issue(testMiningBlock(100, "miner-a")); err != nil {
		t.Fatalf("activation-height work rejected: %v", err)
	}
}
