package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestSetLocalNetworkIDSelectsMainnetHandshake(t *testing.T) {
	t.Cleanup(ResetLocalNetworkIDForTests)

	SetLocalNetworkID(params.NetworkMainnet)
	if LocalNetworkID() != MainnetNetworkID {
		t.Fatalf("local network = %q", LocalNetworkID())
	}

	raw, err := NewHandshakeMessage("node-a", "127.0.0.1:18444", 0, "tip", "0")
	if err != nil {
		t.Fatal(err)
	}
	message, err := DecodeMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	handshake, err := DecodeHandshake(message)
	if err != nil {
		t.Fatal(err)
	}
	if handshake.NetworkID != MainnetNetworkID {
		t.Fatalf("handshake network = %q", handshake.NetworkID)
	}
}

func TestSetLocalNetworkIDDefaultsToPublicTestnet(t *testing.T) {
	ResetLocalNetworkIDForTests()
	if LocalNetworkID() != NetworkID {
		t.Fatalf("local network = %q", LocalNetworkID())
	}
}
