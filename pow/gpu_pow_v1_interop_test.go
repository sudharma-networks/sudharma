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
}

func TestGPUV1InteroperabilityVectorsMatchReference(t *testing.T) {
	raw, err := os.ReadFile("../docs/gpu-pow-v1-interoperability-vectors.json")
	if err != nil {
		t.Fatalf("read interoperability fixture: %v", err)
	}

	var fixture gpuV1InteropFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse interoperability fixture: %v", err)
	}
	if fixture.Algorithm != "sudharma-gpu-pow-v1-reference" {
		t.Fatalf("unexpected algorithm identifier: %q", fixture.Algorithm)
	}
	if len(fixture.Vectors) < 6 {
		t.Fatalf("interoperability corpus too small: got %d vectors", len(fixture.Vectors))
	}

	seenDigests := make(map[string]string, len(fixture.Vectors))
	for _, vector := range fixture.Vectors {
		header, err := hex.DecodeString(vector.HeaderHex)
		if err != nil {
			t.Fatalf("%s: invalid header hex: %v", vector.Name, err)
		}
		nonce, err := strconv.ParseUint(vector.NonceHex, 16, 64)
		if err != nil {
			t.Fatalf("%s: invalid nonce hex: %v", vector.Name, err)
		}
		if vector.CacheNodes == 0 {
			t.Fatalf("%s: cache_nodes must be positive", vector.Name)
		}
		if len(vector.DigestHex) != 64 {
			t.Fatalf("%s: digest must contain 32 bytes", vector.Name)
		}

		cache := GPUV1BuildCache(
			GPUV1EpochSeed(GPUV1EpochForHeight(vector.Height)),
			vector.CacheNodes,
		)
		got := gpuV1ReferenceDigest(header, nonce, vector.Height, cache)
		if gotHex := hex.EncodeToString(got[:]); gotHex != vector.DigestHex {
			t.Fatalf("%s: digest mismatch: got %s want %s", vector.Name, gotHex, vector.DigestHex)
		}
		if prior, exists := seenDigests[vector.DigestHex]; exists {
			t.Fatalf("%s and %s unexpectedly share digest %s", prior, vector.Name, vector.DigestHex)
		}
		seenDigests[vector.DigestHex] = vector.Name
	}
}

func TestGPUV1InteroperabilityVectorsCoverBoundaries(t *testing.T) {
	raw, err := os.ReadFile("../docs/gpu-pow-v1-interoperability-vectors.json")
	if err != nil {
		t.Fatalf("read interoperability fixture: %v", err)
	}

	var fixture gpuV1InteropFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse interoperability fixture: %v", err)
	}

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
			t.Fatalf("interoperability corpus does not cover height %d", height)
		}
	}
	if !hasMaxNonce {
		t.Fatal("interoperability corpus does not cover maximum uint64 nonce")
	}
}
