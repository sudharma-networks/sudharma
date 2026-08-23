package testnet

import "testing"

func TestTestnetIdentityIsStableAndNonEmpty(t *testing.T) {
	if ProtocolNetworkID == "" { t.Fatal("protocol network ID is empty") }
	first := GenesisHash()
	second := GenesisHash()
	if first == "" || first != second { t.Fatalf("unstable genesis hash: %q %q", first, second) }
}
