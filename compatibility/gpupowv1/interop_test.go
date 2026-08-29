package gpupowv1

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

type interoperabilityFixture struct {
	Vectors []struct {
		Name       string `json:"name"`
		HeaderHex  string `json:"header_hex"`
		NonceHex   string `json:"nonce_hex"`
		Height     uint64 `json:"height"`
		CacheNodes uint32 `json:"cache_nodes"`
		DigestHex  string `json:"digest_hex"`
	} `json:"vectors"`
}

func TestIndependentDigestMatchesLockedVectors(t *testing.T) {
	raw, err := os.ReadFile("../../docs/gpu-pow-v1-interoperability-vectors.json")
	if err != nil {
		t.Fatalf("read interoperability fixture: %v", err)
	}

	var fixture interoperabilityFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse interoperability fixture: %v", err)
	}
	if len(fixture.Vectors) < 6 {
		t.Fatalf("interoperability corpus too small: %d", len(fixture.Vectors))
	}

	for _, vector := range fixture.Vectors {
		header, err := hex.DecodeString(vector.HeaderHex)
		if err != nil {
			t.Fatalf("%s: invalid header: %v", vector.Name, err)
		}
		nonce, err := strconv.ParseUint(vector.NonceHex, 16, 64)
		if err != nil {
			t.Fatalf("%s: invalid nonce: %v", vector.Name, err)
		}

		got := Digest(header, nonce, vector.Height, vector.CacheNodes)
		if gotHex := hex.EncodeToString(got[:]); gotHex != vector.DigestHex {
			t.Fatalf("%s: independent digest mismatch: got %s want %s", vector.Name, gotHex, vector.DigestHex)
		}
	}
}
