package p2p

import (
	"encoding/json"
	"testing"
)

func TestHandshakeEncodeDecode(t *testing.T) {
	data, err :=
		NewHandshakeMessage(
			"node-a",
			"127.0.0.1:18444",
			25,
			"test-tip-hash",
			"98765432101234567890",
		)

	if err != nil {
		t.Fatal(err)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		t.Fatal(err)
	}

	handshake, err :=
		DecodeHandshake(message)

	if err != nil {
		t.Fatal(err)
	}

	if handshake.NodeID != "node-a" {
		t.Fatalf(
			"expected node-a, got %s",
			handshake.NodeID,
		)
	}

	if handshake.ListenAddress !=
		"127.0.0.1:18444" {

		t.Fatalf(
			"unexpected listen address: %s",
			handshake.ListenAddress,
		)
	}

	if handshake.Height != 25 {
		t.Fatalf(
			"expected height 25, got %d",
			handshake.Height,
		)
	}

	if handshake.TipHash !=
		"test-tip-hash" {

		t.Fatalf(
			"unexpected tip hash: %s",
			handshake.TipHash,
		)
	}

	if handshake.TotalWork !=
		"98765432101234567890" {

		t.Fatalf(
			"unexpected total work: %s",
			handshake.TotalWork,
		)
	}
}

func TestWrongNetworkRejected(t *testing.T) {
	payload, err :=
		json.Marshal(
			Handshake{
				ProtocolVersion: ProtocolVersion,
				NetworkID:       "fake-network",
				NodeID:          "attacker",
				ListenAddress:   "127.0.0.1:9999",
				Height:          0,
				TipHash:         "fake",
				TotalWork:       "1",
			},
		)

	if err != nil {
		t.Fatal(err)
	}

	message :=
		&Message{
			Type:    MessageHandshake,
			Payload: payload,
		}

	if _, err :=
		DecodeHandshake(
			message,
		); err == nil {

		t.Fatal(
			"handshake from wrong network was accepted",
		)
	}
}

func TestWrongProtocolVersionRejected(t *testing.T) {
	payload, err :=
		json.Marshal(
			Handshake{
				ProtocolVersion: ProtocolVersion + 1,
				NetworkID:       NetworkID,
				NodeID:          "future-node",
				ListenAddress:   "127.0.0.1:18445",
				Height:          0,
				TipHash:         "test",
				TotalWork:       "1",
			},
		)

	if err != nil {
		t.Fatal(err)
	}

	message :=
		&Message{
			Type:    MessageHandshake,
			Payload: payload,
		}

	if _, err :=
		DecodeHandshake(
			message,
		); err == nil {

		t.Fatal(
			"unsupported protocol version was accepted",
		)
	}
}

func TestHandshakeRejectsInvalidTotalWork(t *testing.T) {
	payload, err :=
		json.Marshal(
			Handshake{
				ProtocolVersion: ProtocolVersion,
				NetworkID:       NetworkID,
				NodeID:          "bad-work-node",
				ListenAddress:   "127.0.0.1:18446",
				Height:          10,
				TipHash:         "test",
				TotalWork:       "not-a-number",
			},
		)

	if err != nil {
		t.Fatal(err)
	}

	message :=
		&Message{
			Type:    MessageHandshake,
			Payload: payload,
		}

	if _, err :=
		DecodeHandshake(
			message,
		); err == nil {

		t.Fatal(
			"invalid cumulative work was accepted",
		)
	}
}

func TestHandshakeRejectsNegativeTotalWork(t *testing.T) {
	payload, err :=
		json.Marshal(
			Handshake{
				ProtocolVersion: ProtocolVersion,
				NetworkID:       NetworkID,
				NodeID:          "negative-work-node",
				ListenAddress:   "127.0.0.1:18447",
				Height:          10,
				TipHash:         "test",
				TotalWork:       "-1",
			},
		)

	if err != nil {
		t.Fatal(err)
	}

	message :=
		&Message{
			Type:    MessageHandshake,
			Payload: payload,
		}

	if _, err :=
		DecodeHandshake(
			message,
		); err == nil {

		t.Fatal(
			"negative cumulative work was accepted",
		)
	}
}

func TestNewHandshakeRejectsBadTotalWork(t *testing.T) {
	if _, err :=
		NewHandshakeMessage(
			"node-a",
			"127.0.0.1:18444",
			0,
			"tip",
			"abc",
		); err == nil {

		t.Fatal(
			"invalid total work was accepted",
		)
	}
}
