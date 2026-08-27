package pow

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestHashBlockForVersionKeepsLegacyV1(t *testing.T) {
	b := &blockchain.Block{
		Version:      1,
		Height:       42,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	const nonce uint64 = 0x0102030405060708

	got := HashBlockForVersion(b, nonce)
	want := HashBlock(b, nonce)
	if got != want {
		t.Fatalf("legacy v1 hash changed: got %s want %s", got, want)
	}
}

func TestHashBlockForVersionUsesGPUV2Domain(t *testing.T) {
	b := &blockchain.Block{
		Version:      2,
		Height:       42,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	const nonce uint64 = 0x0102030405060708

	got := HashBlockForVersion(b, nonce)
	legacy := HashBlock(b, nonce)
	if got == legacy {
		t.Fatal("GPU-PoW v1 must be domain-separated from legacy double-SHA-256")
	}
	if got != HashBlockForVersion(b, nonce) {
		t.Fatal("GPU-PoW v1 hash must be deterministic")
	}
}
