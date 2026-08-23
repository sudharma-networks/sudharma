package p2p

import (
	"encoding/json"
	"testing"
)

func TestGetPeersMessageEncodeDecode(t *testing.T) {
	data, err := NewGetPeersMessage()
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if message.Type != MessageGetPeers {
		t.Fatalf(
			"expected %s, got %s",
			MessageGetPeers,
			message.Type,
		)
	}

	if err := DecodeGetPeers(message); err != nil {
		t.Fatal(err)
	}
}

func TestPeersMessageEncodeDecode(t *testing.T) {
	input := []KnownPeer{
		{
			NodeID:  "peer-b",
			Address: "127.0.0.1:19002",
		},
		{
			NodeID:  "peer-c",
			Address: "127.0.0.1:19003",
		},
	}

	data, err := NewPeersMessage(input)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if message.Type != MessagePeers {
		t.Fatalf(
			"expected %s, got %s",
			MessagePeers,
			message.Type,
		)
	}

	decoded, err := DecodePeers(message)
	if err != nil {
		t.Fatal(err)
	}

	if len(decoded) != 2 {
		t.Fatalf(
			"expected 2 peers, got %d",
			len(decoded),
		)
	}

	if decoded[0].NodeID != "peer-b" ||
		decoded[0].Address != "127.0.0.1:19002" {

		t.Fatal("first peer mismatch")
	}

	if decoded[1].NodeID != "peer-c" ||
		decoded[1].Address != "127.0.0.1:19003" {

		t.Fatal("second peer mismatch")
	}
}

func TestPeersMessageDeduplicates(t *testing.T) {
	input := []KnownPeer{
		{
			NodeID:  "peer-b",
			Address: "127.0.0.1:19002",
		},
		{
			NodeID:  "peer-b",
			Address: "127.0.0.1:19999",
		},
		{
			NodeID:  "peer-c",
			Address: "127.0.0.1:19002",
		},
	}

	data, err := NewPeersMessage(input)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodePeers(message)
	if err != nil {
		t.Fatal(err)
	}

	if len(decoded) != 1 {
		t.Fatalf(
			"expected duplicate entries to collapse to 1, got %d",
			len(decoded),
		)
	}
}

func TestPeersMessageRejectsEmptyNodeID(t *testing.T) {
	_, err := NewPeersMessage(
		[]KnownPeer{
			{
				NodeID:  "",
				Address: "127.0.0.1:19002",
			},
		},
	)

	if err == nil {
		t.Fatal(
			"expected empty peer node ID to be rejected",
		)
	}
}

func TestPeersMessageRejectsEmptyAddress(t *testing.T) {
	_, err := NewPeersMessage(
		[]KnownPeer{
			{
				NodeID:  "peer-b",
				Address: "",
			},
		},
	)

	if err == nil {
		t.Fatal(
			"expected empty peer address to be rejected",
		)
	}
}

func TestPeersMessageRejectsOversizedPayload(t *testing.T) {
	peers := make(
		[]KnownPeer,
		MaxPeersPerMessage+1,
	)

	for i := range peers {
		peers[i] = KnownPeer{
			NodeID:  "peer-" + string(rune(i+1000)),
			Address: "127.0.0.1:19000",
		}
	}

	_, err := NewPeersMessage(peers)

	if err == nil {
		t.Fatal(
			"expected oversized peer message to be rejected",
		)
	}
}

func TestDecodePeersRejectsMalformedEntry(t *testing.T) {
	payload, err := json.Marshal(
		PeersMessage{
			Peers: []KnownPeer{
				{
					NodeID:  "peer-b",
					Address: "",
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	message := &Message{
		Type:    MessagePeers,
		Payload: payload,
	}

	_, err = DecodePeers(message)

	if err == nil {
		t.Fatal(
			"expected malformed discovered peer to be rejected",
		)
	}
}
