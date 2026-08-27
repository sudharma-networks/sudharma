package blockchain

import (
	"encoding/hex"
	"testing"
)

// TestLegacyPoWHeaderBaseline freezes the pre-GPU-PoW canonical header
// serialization. GPU-PoW activation must be versioned and must not silently
// reinterpret blocks produced under the legacy rule.
func TestLegacyPoWHeaderBaseline(t *testing.T) {
	b := &Block{
		Version:      1,
		Height:       42,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}

	const nonce uint64 = 0x0102030405060708
	got := hex.EncodeToString(b.HeaderBytes(nonce))
	const want = "00000001000000000000002a000000006a80fdbc303132333435363738396162636465666665646362613938373635343332313039636364633039343438393837346265643838386666653462646639623832393866346335313331000000070102030405060708"

	if got != want {
		t.Fatalf("legacy header changed:\n got %s\nwant %s", got, want)
	}
}
