package pow

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestHashBlockForVersionWithCacheUsesGPUV1Reference(t *testing.T) {
	b := &blockchain.Block{
		Version:      2,
		Height:       7500,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   1,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	const nonce uint64 = 0x0123456789abcdef
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(b.Height)), 8)

	want := gpuV1HashBlockWithCache(b, nonce, cache)
	if got := HashBlockForVersionWithCache(b, nonce, cache); got != want {
		t.Fatalf("version-aware cached hash mismatch: got %s want %s", got, want)
	}
}

func TestHashBlockForVersionWithCachePreservesLegacyV1(t *testing.T) {
	b := blockchain.NewGenesisBlock()
	b.Version = 1

	want := HashBlock(b, 42)
	if got := HashBlockForVersionWithCache(b, 42, nil); got != want {
		t.Fatalf("legacy hash changed: got %s want %s", got, want)
	}
}

func TestHashBlockForVersionRejectsImplicitV2Cache(t *testing.T) {
	b := &blockchain.Block{Version: 2}
	if got := HashBlockForVersion(b, 0); got != "" {
		t.Fatalf("version 2 must not use an implicit/scaffold cache path: got %s", got)
	}
}

func TestCheckBlockWithCacheUsesVersionAppropriatePoW(t *testing.T) {
	b := &blockchain.Block{
		Version:      2,
		Height:       7500,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   1,
		Nonce:        99,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(b.Height)), 8)

	if !CheckBlockWithCache(b, cache) {
		t.Fatal("difficulty-1 version-2 block should validate with explicit cache")
	}
	if CheckBlock(b) {
		t.Fatal("generic CheckBlock must not silently validate version 2 before activation/cache policy is frozen")
	}
}
