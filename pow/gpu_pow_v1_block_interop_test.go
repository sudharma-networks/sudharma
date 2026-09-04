package pow

import (
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func blockFromGPUV1InteropVector(t *testing.T, vector gpuV1BlockInteropVector) *blockchain.Block {
	t.Helper()
	nonce, err := strconv.ParseUint(vector.NonceHex, 16, 64)
	if err != nil {
		t.Fatalf("%s: invalid nonce: %v", vector.Name, err)
	}
	return &blockchain.Block{
		Version:      vector.Version,
		Height:       vector.Height,
		Timestamp:    vector.Timestamp,
		PreviousHash: vector.PreviousHash,
		MerkleRoot:   vector.MerkleRoot,
		Difficulty:   vector.Difficulty,
		Nonce:        nonce,
		MinerAddress: vector.MinerAddress,
	}
}

func TestGPUV1BlockInteroperabilityVectorMatchesCanonicalHeaderAndDigest(t *testing.T) {
	fixture := loadGPUV1InteropFixture(t)
	if len(fixture.BlockVectors) != 1 {
		t.Fatalf("block vector count = %d want 1", len(fixture.BlockVectors))
	}
	vector := fixture.BlockVectors[0]
	block := blockFromGPUV1InteropVector(t, vector)
	if vector.CacheNodes == 0 {
		t.Fatal("cache_nodes must be positive")
	}

	headerWithZeroNonce := block.HeaderBytes(0)
	if len(headerWithZeroNonce) < 8 {
		t.Fatal("canonical block header is too short")
	}
	prefix := hex.EncodeToString(headerWithZeroNonce[:len(headerWithZeroNonce)-8])
	if prefix != vector.HeaderPrefix {
		t.Fatalf("header prefix = %s want %s", prefix, vector.HeaderPrefix)
	}

	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(block.Height)), vector.CacheNodes)
	if got := GPUV1HashBlockWithCache(block, block.Nonce, cache); got != vector.DigestHex {
		t.Fatalf("block digest = %s want %s", got, vector.DigestHex)
	}
}

func TestGPUV1BlockDigestBindsEveryConsensusHeaderField(t *testing.T) {
	vector := loadGPUV1InteropFixture(t).BlockVectors[0]
	base := blockFromGPUV1InteropVector(t, vector)
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(base.Height)), vector.CacheNodes)
	want := GPUV1HashBlockWithCache(base, base.Nonce, cache)

	mutations := []struct {
		name string
		edit func(*blockchain.Block)
	}{
		{"nonce", func(b *blockchain.Block) { b.Nonce++ }},
		{"version", func(b *blockchain.Block) { b.Version++ }},
		{"height", func(b *blockchain.Block) { b.Height++ }},
		{"timestamp", func(b *blockchain.Block) { b.Timestamp++ }},
		{"previous-hash", func(b *blockchain.Block) { b.PreviousHash = "1123456789abcdef" }},
		{"merkle-root", func(b *blockchain.Block) { b.MerkleRoot = "eedcba9876543210" }},
		{"difficulty", func(b *blockchain.Block) { b.Difficulty++ }},
		{"miner-address", func(b *blockchain.Block) { b.MinerAddress = "bb90e4a17bbd907135f37610766e27c70a95a728" }},
	}

	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			mutated := *base
			tc.edit(&mutated)
			mutatedCache := cache
			if GPUV1EpochForHeight(mutated.Height) != GPUV1EpochForHeight(base.Height) {
				mutatedCache = GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(mutated.Height)), vector.CacheNodes)
			}
			if got := GPUV1HashBlockWithCache(&mutated, mutated.Nonce, mutatedCache); got == want {
				t.Fatalf("%s mutation reproduced canonical digest %s", tc.name, want)
			}
		})
	}
}

func TestGPUV1BlockReferenceHelpersFailClosed(t *testing.T) {
	if got := GPUV1HashBlockWithCache(nil, 1, make([]GPUV1CacheNode, 1)); got != "" {
		t.Fatalf("nil block hash = %q want empty", got)
	}
	block := &blockchain.Block{Version: 2, Height: 1, Difficulty: 1, Nonce: 7}
	if got := GPUV1HashBlockWithCache(block, block.Nonce, nil); got != "" {
		t.Fatalf("empty cache hash = %q want empty", got)
	}
	if GPUV1CheckBlockWithCache(nil, make([]GPUV1CacheNode, 1)) {
		t.Fatal("nil block was accepted")
	}
	if GPUV1CheckBlockWithCache(block, nil) {
		t.Fatal("empty cache was accepted")
	}
	wrongVersion := *block
	wrongVersion.Version = 1
	if GPUV1CheckBlockWithCache(&wrongVersion, make([]GPUV1CacheNode, 1)) {
		t.Fatal("non-Version-2 block was accepted by GPU verifier")
	}
	zeroDifficulty := *block
	zeroDifficulty.Difficulty = 0
	if GPUV1CheckBlockWithCache(&zeroDifficulty, make([]GPUV1CacheNode, 1)) {
		t.Fatal("difficulty zero was accepted")
	}
}

func TestGPUV1CheckBlockWithCacheAcceptsMaximumTargetReferenceProof(t *testing.T) {
	block := &blockchain.Block{
		Version:      2,
		Height:       7500,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   1,
		Nonce:        99,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(block.Height)), 8)
	if !GPUV1CheckBlockWithCache(block, cache) {
		t.Fatal("difficulty-1 Version-2 reference proof was rejected")
	}
}
