package pow

import (
	"encoding/hex"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestGPUV1ReferenceDigestVector(t *testing.T) {
	header := []byte("sudharma-gpu-pow-v1-reference-header")
	nonce := uint64(0x0123456789abcdef)
	height := uint64(22501)
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(height)), 8)

	got := gpuV1ReferenceDigest(header, nonce, height, cache)
	want := [32]byte{
		0xb5, 0xd8, 0xc0, 0xc4, 0x58, 0xd3, 0x7e, 0xb5,
		0x51, 0xe8, 0xe8, 0x45, 0x48, 0x8f, 0xec, 0x75,
		0x6c, 0xa0, 0x2c, 0x8e, 0xba, 0x89, 0x1b, 0xc1,
		0x04, 0xc4, 0xf9, 0xc4, 0x26, 0xc9, 0x61, 0x70,
	}
	if got != want {
		t.Fatalf("reference digest vector mismatch: got %x want %x", got, want)
	}
}

func TestGPUV1ReferenceDigestDeterministic(t *testing.T) {
	header := []byte("deterministic-header")
	height := uint64(7503)
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(height)), 8)

	a := gpuV1ReferenceDigest(header, 42, height, cache)
	b := gpuV1ReferenceDigest(header, 42, height, cache)
	if a != b {
		t.Fatal("reference digest is not deterministic")
	}
}

func TestGPUV1ReferenceDigestDependsOnNonceAndHeight(t *testing.T) {
	header := []byte("sensitivity-header")
	cache0 := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(0)), 8)
	cache1 := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(GPUV1EpochLength)), 8)

	base := gpuV1ReferenceDigest(header, 7, 0, cache0)
	if got := gpuV1ReferenceDigest(header, 8, 0, cache0); got == base {
		t.Fatal("different nonce produced identical reference digest")
	}
	if got := gpuV1ReferenceDigest(header, 7, GPUV1EpochLength, cache1); got == base {
		t.Fatal("different height/epoch produced identical reference digest")
	}
}

func TestGPUV1HashBlockWithCacheMatchesCanonicalReference(t *testing.T) {
	b := &blockchain.Block{
		Version:      2,
		Height:       22501,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	const nonce uint64 = 0x0123456789abcdef
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(b.Height)), 8)

	headerWithZeroNonce := b.HeaderBytes(0)
	headerPrefix := headerWithZeroNonce[:len(headerWithZeroNonce)-8]
	wantDigest := gpuV1ReferenceDigest(headerPrefix, nonce, b.Height, cache)
	want := hex.EncodeToString(wantDigest[:])

	if got := gpuV1HashBlockWithCache(b, nonce, cache); got != want {
		t.Fatalf("block reference hash mismatch: got %s want %s", got, want)
	}
}

func TestGPUV1CheckBlockWithCacheUsesTarget(t *testing.T) {
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

	if !gpuV1CheckBlockWithCache(b, cache) {
		t.Fatal("difficulty-1 GPU-PoW reference block should satisfy the maximum target")
	}

	b.Difficulty = 0
	if gpuV1CheckBlockWithCache(b, cache) {
		t.Fatal("difficulty zero must be rejected")
	}
}
