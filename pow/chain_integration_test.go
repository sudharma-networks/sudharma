package pow

import (
	"strconv"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func TestKhushiFinitePolicyIntegratesWithBlockchainAtActivationBoundary(t *testing.T) {
	const activation uint64 = 3
	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: activation}
	verifier, err := newChainProofVerifier(policy, 8)
	if err != nil {
		t.Fatal(err)
	}
	chain, err := blockchain.NewChainForWithConsensus(
		params.NetworkPublicTestnet,
		policy,
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	genesis := chain.Tip()
	if genesis == nil {
		t.Fatal("test chain missing genesis")
	}

	block1 := newIntegrationBlock(t, chain, 1, 1, genesis.Timestamp+1)
	mineLegacyIntegrationBlock(t, block1)
	if err := chain.AddBlock(block1); err != nil {
		t.Fatalf("add pre-activation Version 1 block at height 1: %v", err)
	}

	block2 := newIntegrationBlock(t, chain, 1, 2, block1.Timestamp+1)
	mineLegacyIntegrationBlock(t, block2)

	preActivationReplay := cloneIntegrationBlock(block2)
	preActivationReplay.Version = 2
	preActivationReplay.Nonce = 0
	if err := chain.AddBlock(preActivationReplay); err == nil {
		t.Fatal("Version 2 replay was accepted before activation")
	}
	if chain.Height() != 1 {
		t.Fatalf("pre-activation rejection mutated chain height to %d", chain.Height())
	}

	if err := chain.AddBlock(block2); err != nil {
		t.Fatalf("add pre-activation Version 1 block at height 2: %v", err)
	}

	legacyAfterActivation := newIntegrationBlock(t, chain, 1, activation, block2.Timestamp+1)
	mineLegacyIntegrationBlock(t, legacyAfterActivation)
	if err := chain.AddBlock(legacyAfterActivation); err == nil {
		t.Fatal("Version 1 block was accepted at Khushi activation height")
	}
	if chain.Height() != activation-1 {
		t.Fatalf("post-activation Version 1 rejection mutated chain height to %d", chain.Height())
	}

	validV2 := newIntegrationBlock(t, chain, 2, activation, block2.Timestamp+1)
	cache := verifier.cacheForHeight(validV2.Height)
	mineKhushiIntegrationBlock(t, validV2, cache)

	badNonce := invalidNonceMutation(t, verifier, validV2)
	if err := chain.AddBlock(badNonce); err == nil {
		t.Fatal("Khushi block with an invalid nonce was accepted")
	}
	if chain.Height() != activation-1 {
		t.Fatalf("invalid-nonce rejection mutated chain height to %d", chain.Height())
	}

	badHeader := invalidMinerMutation(t, verifier, validV2)
	if err := chain.AddBlock(badHeader); err == nil {
		t.Fatal("Khushi block with a mutated header and stale proof was accepted")
	}
	if chain.Height() != activation-1 {
		t.Fatalf("header-mutation rejection mutated chain height to %d", chain.Height())
	}

	if err := chain.AddBlock(validV2); err != nil {
		t.Fatalf("valid Version 2 block rejected at activation boundary: %v", err)
	}
	if chain.Height() != activation {
		t.Fatalf("chain height = %d want %d", chain.Height(), activation)
	}
}

func TestDefaultPublicTestnetChainRemainsActivationDisabled(t *testing.T) {
	chain, err := blockchain.NewChainFor(params.NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	policy := chain.PoWPolicy()
	if policy.GPUV1ActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf(
			"default public-testnet GPU activation = %d want disabled sentinel %d",
			policy.GPUV1ActivationHeight,
			params.GPUV1ActivationDisabled,
		)
	}

	candidate := newIntegrationBlock(t, chain, 2, 1, chain.Tip().Timestamp+1)
	candidate.Nonce = 0
	if err := chain.AddBlock(candidate); err == nil {
		t.Fatal("default public-testnet constructor accepted Version 2 while activation is disabled")
	}
	if chain.Height() != 0 {
		t.Fatalf("default Version-2 rejection mutated chain height to %d", chain.Height())
	}
}

func newIntegrationBlock(
	t *testing.T,
	chain *blockchain.Chain,
	version uint32,
	height uint64,
	timestamp int64,
) *blockchain.Block {
	t.Helper()
	if chain == nil || chain.Tip() == nil {
		t.Fatal("cannot build integration block without chain tip")
	}
	difficulty, err := blockchain.ExpectedNextDifficulty(chain)
	if err != nil {
		t.Fatal(err)
	}
	block := &blockchain.Block{
		Version:      version,
		Height:       height,
		Timestamp:    timestamp,
		PreviousHash: chain.Tip().Hash(),
		Difficulty:   difficulty,
		MinerAddress: "khushi-integration-miner",
		Transactions: nil,
	}
	block.UpdateMerkleRoot()
	return block
}

func mineLegacyIntegrationBlock(t *testing.T, block *blockchain.Block) {
	t.Helper()
	for nonce := uint64(0); nonce < 100_000; nonce++ {
		block.Nonce = nonce
		if CheckBlock(block) {
			return
		}
	}
	t.Fatalf("failed to find legacy test proof at difficulty %d", block.Difficulty)
}

func mineKhushiIntegrationBlock(t *testing.T, block *blockchain.Block, cache []GPUV1CacheNode) {
	t.Helper()
	for nonce := uint64(0); nonce < 10_000; nonce++ {
		block.Nonce = nonce
		if GPUV1CheckBlockWithCache(block, cache) {
			return
		}
	}
	t.Fatalf("failed to find compact Khushi test proof at difficulty %d", block.Difficulty)
}

func invalidNonceMutation(
	t *testing.T,
	verifier *chainProofVerifier,
	valid *blockchain.Block,
) *blockchain.Block {
	t.Helper()
	for delta := uint64(1); delta < 1_000; delta++ {
		candidate := cloneIntegrationBlock(valid)
		candidate.Nonce = valid.Nonce + delta
		if !verifier.Verify(candidate) {
			return candidate
		}
	}
	t.Fatal("failed to find a deterministic invalid nonce mutation")
	return nil
}

func invalidMinerMutation(
	t *testing.T,
	verifier *chainProofVerifier,
	valid *blockchain.Block,
) *blockchain.Block {
	t.Helper()
	for i := 0; i < 1_000; i++ {
		candidate := cloneIntegrationBlock(valid)
		candidate.MinerAddress = "khushi-mutated-miner-" + strconv.Itoa(i)
		if !verifier.Verify(candidate) {
			return candidate
		}
	}
	t.Fatal("failed to find a deterministic invalid header mutation")
	return nil
}

func cloneIntegrationBlock(block *blockchain.Block) *blockchain.Block {
	if block == nil {
		return nil
	}
	clone := *block
	if block.Transactions != nil {
		clone.Transactions = append(clone.Transactions[:0:0], block.Transactions...)
	}
	return &clone
}
