package pow

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestChainProofVerifierUsesProductionPolicy(t *testing.T) {
	if productionGPUV1CacheNodeCount != 262_144 {
		t.Fatalf("production cache nodes = %d, want 262144", productionGPUV1CacheNodeCount)
	}
	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: 100}
	verifier, err := NewChainProofVerifier(policy)
	if err != nil {
		t.Fatal(err)
	}
	if !verifier.SupportsVersion(1) || !verifier.SupportsVersion(2) {
		t.Fatal("chain verifier must support legacy and GPU-PoW versions")
	}
	if verifier.SupportsVersion(3) {
		t.Fatal("chain verifier reported support for a future version")
	}
}

func TestChainProofVerifierDispatchesWithoutFallback(t *testing.T) {
	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: 1}
	verifier, err := NewChainProofVerifier(policy)
	if err != nil {
		t.Fatal(err)
	}

	legacy := &blockchain.Block{Version: 1, Height: 0, Difficulty: 1}
	if !verifier.Verify(legacy) {
		t.Fatal("valid legacy proof was rejected")
	}
	gpu := &blockchain.Block{Version: 2, Height: 1, Difficulty: 1}
	if !verifier.Verify(gpu) {
		t.Fatal("valid GPU-PoW proof was rejected")
	}
	gpu.Difficulty = 0
	if verifier.Verify(gpu) {
		t.Fatal("invalid GPU-PoW proof fell back to another algorithm")
	}
	if verifier.Verify(&blockchain.Block{Version: 3, Height: 1, Difficulty: 1}) {
		t.Fatal("future block version was verified")
	}
}

func TestChainProofVerifierRetainsOnlyCurrentEpoch(t *testing.T) {
	verifier, err := NewChainProofVerifier(blockchain.PoWPolicy{GPUV1ActivationHeight: 1})
	if err != nil {
		t.Fatal(err)
	}
	implementation, ok := verifier.(*chainProofVerifier)
	if !ok {
		t.Fatalf("verifier type = %T, want *chainProofVerifier", verifier)
	}

	if !verifier.Verify(&blockchain.Block{Version: 2, Height: 1, Difficulty: 1}) {
		t.Fatal("epoch zero proof was rejected")
	}
	if got := implementation.cachedEpochCount(); got != 1 {
		t.Fatalf("cached epochs = %d, want 1", got)
	}
	if !verifier.Verify(&blockchain.Block{Version: 2, Height: GPUV1EpochLength, Difficulty: 1}) {
		t.Fatal("next epoch proof was rejected")
	}
	if got := implementation.cachedEpochCount(); got != 1 {
		t.Fatalf("cached epochs after rollover = %d, want 1", got)
	}
}
