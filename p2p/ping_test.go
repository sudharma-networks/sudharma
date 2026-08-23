package p2p

import "testing"

func TestPingEncodeDecode(t *testing.T) {
	const expectedNonce uint64 = 123456789

	data, err := NewPingMessage(expectedNonce)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	ping, err := DecodePing(message)
	if err != nil {
		t.Fatal(err)
	}

	if ping.Nonce != expectedNonce {
		t.Fatalf(
			"expected ping nonce %d, got %d",
			expectedNonce,
			ping.Nonce,
		)
	}
}

func TestPongEncodeDecode(t *testing.T) {
	const expectedNonce uint64 = 987654321

	data, err := NewPongMessage(expectedNonce)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	pong, err := DecodePong(message)
	if err != nil {
		t.Fatal(err)
	}

	if pong.Nonce != expectedNonce {
		t.Fatalf(
			"expected pong nonce %d, got %d",
			expectedNonce,
			pong.Nonce,
		)
	}
}

func TestPingCannotDecodeAsPong(t *testing.T) {
	data, err := NewPingMessage(123)
	if err != nil {
		t.Fatal(err)
	}

	message, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodePong(message); err == nil {
		t.Fatal(
			"ping message was incorrectly accepted as pong",
		)
	}
}

func TestUnknownMessageRejected(t *testing.T) {
	data := []byte(
		`{"type":"fake-message","payload":{}}`,
	)

	if _, err := DecodeMessage(data); err == nil {
		t.Fatal(
			"unknown P2P message type was accepted",
		)
	}
}
