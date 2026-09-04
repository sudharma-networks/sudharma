package pow

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

type gpuV1InteropFixture struct {
	Algorithm string `json:"algorithm"`
	Vectors   []struct {
		Name       string `json:"name"`
		HeaderHex  string `json:"header_hex"`
		NonceHex   string `json:"nonce_hex"`
		Height     uint64 `json:"height"`
		CacheNodes uint32 `json:"cache_nodes"`
		DigestHex  string `json:"digest_hex"`
	} `json:"vectors"`
	BlockVectors []gpuV1BlockInteropVector `json:"block_vectors"`
}

type gpuV1BlockInteropVector struct {
	Name         string `json:"name"`
	Version      uint32 `json:"version"`
	Height       uint64 `json:"height"`
	Timestamp    int64  `json:"timestamp"`
	PreviousHash string `json:"previous_hash"`
	MerkleRoot   string `json:"merkle_root"`
	Difficulty   uint32 `json:"difficulty"`
	MinerAddress string `json:"miner_address"`
	NonceHex     string `json:"nonce_hex"`
	CacheNodes   uint32 `json:"cache_nodes"`
	HeaderPrefix string `json:"header_prefix_hex"`
	DigestHex    string `json:"digest_hex"`
}

func loadGPUV1InteropFixture(t *testing.T) gpuV1InteropFixture {
	t.Helper()
	raw, err := os.ReadFile("../docs/gpu-pow-v1-interoperability-vectors.json")
	if err != nil {
		t.Fatalf("read interoperability fixture: %v", err)
	}
	var fixture gpuV1InteropFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse interoperability fixture: %v", err)
	}
	if fixture.Algorithm != GPUV1AlgorithmID {
		t.Fatalf("algorithm = %q want %q", fixture.Algorithm, GPUV1AlgorithmID)
	}
	return fixture
}

func TestGPUV1InteroperabilityVectorsMatchReference(t *testing.T) {
	fixture := loadGPUV1InteropFixture(t)
	if len(fixture.Vectors) != 6 {
		t.Fatalf("vector count = %d want 6", len(fixture.Vectors))
	}

	seen := make(map[string]bool, len(fixture.Vectors))
	for _, vector := range fixture.Vectors {
		header, err := hex.DecodeString(vector.HeaderHex)
		if err != nil {
			t.Fatalf("%s: invalid header hex: %v", vector.Name, err)
		}
		nonce, err := strconv.ParseUint(vector.NonceHex, 16, 64)
		if err != nil {
			t.Fatalf("%s: invalid nonce: %v", vector.Name, err)
		}
		if vector.CacheNodes == 0 {
			t.Fatalf("%s: cache_nodes must be positive", vector.Name)
		}
		cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(vector.Height)), vector.CacheNodes)
		got := GPUV1ReferenceDigest(header, nonce, vector.Height, cache)
		if gotHex := hex.EncodeToString(got[:]); gotHex != vector.DigestHex {
			t.Fatalf("%s: digest = %s want %s", vector.Name, gotHex, vector.DigestHex)
		}
		if seen[vector.DigestHex] {
			t.Fatalf("duplicate locked digest %s", vector.DigestHex)
		}
		seen[vector.DigestHex] = true
	}
}

func TestGPUV1InteroperabilityVectorsCoverProgramAndEpochBoundaries(t *testing.T) {
	fixture := loadGPUV1InteropFixture(t)
	heights := make(map[uint64]bool, len(fixture.Vectors))
	hasMaxNonce := false
	for _, vector := range fixture.Vectors {
		heights[vector.Height] = true
		if vector.NonceHex == "ffffffffffffffff" {
			hasMaxNonce = true
		}
	}
	for _, height := range []uint64{0, 2, 3, GPUV1EpochLength - 1, GPUV1EpochLength} {
		if !heights[height] {
			t.Fatalf("missing boundary vector at height %d", height)
		}
	}
	if !hasMaxNonce {
		t.Fatal("maximum uint64 nonce is not covered")
	}
}

func TestGPUV1ReferenceDigestFailsClosedOnEmptyCache(t *testing.T) {
	got := GPUV1ReferenceDigest([]byte("header"), 1, 0, nil)
	if got != ([32]byte{}) {
		t.Fatalf("empty-cache digest = %x want zero", got)
	}
}

func TestGPUV1ReferenceDigestRejectsWrongEpochByDigestMismatch(t *testing.T) {
	header := []byte("epoch-bound-header")
	height := GPUV1EpochLength
	good := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(height)), 8)
	wrong := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(height)-1), 8)
	if GPUV1ReferenceDigest(header, 7, height, good) == GPUV1ReferenceDigest(header, 7, height, wrong) {
		t.Fatal("wrong epoch cache reproduced the canonical digest")
	}
}
