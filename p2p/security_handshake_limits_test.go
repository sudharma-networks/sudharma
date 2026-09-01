package p2p

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeHandshakeRejectsOversizedTotalWork(t *testing.T) {
	payload, err := json.Marshal(Handshake{
		ProtocolVersion: ProtocolVersion,
		NetworkID:       LocalNetworkID(),
		NodeID:          "oversized-work-peer",
		ListenAddress:   "127.0.0.1:28444",
		Height:          1,
		TipHash:         strings.Repeat("0", 64),
		TotalWork:       strings.Repeat("9", MaxHandshakeTotalWorkDigits+1),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecodeHandshake(&Message{Type: MessageHandshake, Payload: payload})
	if err == nil {
		t.Fatal("oversized total_work must be rejected before big.Int parsing")
	}
}

func TestNewHandshakeMessageRejectsOversizedTotalWork(t *testing.T) {
	_, err := NewHandshakeMessage(
		"local-node",
		"127.0.0.1:28444",
		1,
		strings.Repeat("0", 64),
		strings.Repeat("9", MaxHandshakeTotalWorkDigits+1),
	)
	if err == nil {
		t.Fatal("oversized local total_work must be rejected")
	}
}
