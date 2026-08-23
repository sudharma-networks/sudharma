package testnet

import "testing"

func TestTestnetIdentityIsStableAndNonEmpty(t *testing.T) {
	if ProtocolNetworkID == "" {
		t.Fatal("protocol network ID is empty")
	}
	if ProtocolNetworkID != Slug {
		t.Fatalf("P2P network ID %q does not match testnet slug %q", ProtocolNetworkID, Slug)
	}
	first := GenesisHash()
	second := GenesisHash()
	if first == "" || first != second {
		t.Fatalf("unstable genesis hash: %q %q", first, second)
	}
}
