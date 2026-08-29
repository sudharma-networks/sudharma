package miner

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
)

type activationProofVerifier struct{}

func (activationProofVerifier) SupportsVersion(version uint32) bool {
	return version == 1 || version == 2
}

func (activationProofVerifier) Verify(*blockchain.Block) bool {
	return true
}

func TestMineNextBlockRequiresExternalGPUMinerAfterActivation(t *testing.T) {
	chain, err := blockchain.NewChainWithConsensus(
		blockchain.PoWPolicy{GPUV1ActivationHeight: 1},
		activationProofVerifier{},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = MineNextBlock(
		chain,
		blockchain.NewState(),
		mempool.NewMempool(),
		"gpu-miner",
		1,
	)
	if err == nil || err.Error() != "external GPU miner required" {
		t.Fatalf("MineNextBlock error = %v, want external GPU miner required", err)
	}
}
