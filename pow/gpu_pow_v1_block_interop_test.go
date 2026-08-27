package pow

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

type gpuV1BlockInteropFixture struct {
	BlockVectors []struct {
		Name          string `json:"name"`
		Version       uint32 `json:"version"`
		Height        uint64 `json:"height"`
		Timestamp     int64  `json:"timestamp"`
		PreviousHash  string `json:"previous_hash"`
		MerkleRoot    string `json:"merkle_root"`
		Difficulty    uint32 `json:"difficulty"`
		MinerAddress  string `json:"miner_address"`
		NonceHex      string `json:"nonce_hex"`
		CacheNodes    uint32 `json:"cache_nodes"`
		HeaderPrefix  string `json:"header_prefix_hex"`
		DigestHex     string `json:"digest_hex"`
	} `json:"block_vectors"`
}

func TestGPUV1BlockInteroperabilityVectorMatchesCanonicalHeader(t *testing.T) {
	raw, err := os.ReadFile("../docs/gpu-pow-v1-interoperability-vectors.json")
	if err != nil {
		t.Fatalf("read interoperability fixture: %v", err)
	}

	var fixture gpuV1BlockInteropFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse interoperability fixture: %v", err)
	}
	if len(fixture.BlockVectors) != 1 {
		t.Fatalf("expected exactly one locked block vector, got %d", len(fixture.BlockVectors))
	}

	vector := fixture.BlockVectors[0]
	nonce, err := strconv.ParseUint(vector.NonceHex, 16, 64)
	if err != nil {
		t.Fatalf("%s: invalid nonce: %v", vector.Name, err)
	}
	if vector.CacheNodes == 0 {
		t.Fatalf("%s: cache_nodes must be positive", vector.Name)
	}
	if _, err := hex.DecodeString(vector.HeaderPrefix); err != nil {
		t.Fatalf("%s: invalid header prefix hex: %v", vector.Name, err)
	}
	if len(vector.DigestHex) != 64 {
		t.Fatalf("%s: digest must contain 32 bytes", vector.Name)
	}

	block := &blockchain.Block{
		Version:      vector.Version,
		Height:       vector.Height,
		Timestamp:    vector.Timestamp,
		PreviousHash: vector.PreviousHash,
		MerkleRoot:   vector.MerkleRoot,
		Difficulty:   vector.Difficulty,
		Nonce:        nonce,
		MinerAddress: vector.MinerAddress,
	}

	headerWithZeroNonce := block.HeaderBytes(0)
	if len(headerWithZeroNonce) < 8 {
		t.Fatal("canonical block header is too short to contain nonce")
	}
	gotPrefix := hex.EncodeToString(headerWithZeroNonce[:len(headerWithZeroNonce)-8])
	if gotPrefix != vector.HeaderPrefix {
		t.Fatalf("%s: canonical header prefix mismatch: got %s want %s", vector.Name, gotPrefix, vector.HeaderPrefix)
	}

	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(block.Height)), vector.CacheNodes)
	if got := gpuV1HashBlockWithCache(block, nonce, cache); got != vector.DigestHex {
		t.Fatalf("%s: canonical block digest mismatch: got %s want %s", vector.Name, got, vector.DigestHex)
	}
}

func TestGPUV1BlockInteroperabilityHashBindsMinerAddress(t *testing.T) {
	block := &blockchain.Block{
		Version:      2,
		Height:       22501,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		Nonce:        0x0123456789abcdef,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(block.Height)), 8)
	base := gpuV1HashBlockWithCache(block, block.Nonce, cache)

	mutated := *block
	mutated.MinerAddress = "bb90e4a17bbd907135f37610766e27c70a95a728"
	if got := gpuV1HashBlockWithCache(&mutated, mutated.Nonce, cache); got == base {
		t.Fatal("changing miner address did not change GPU-PoW block digest")
	}
}
