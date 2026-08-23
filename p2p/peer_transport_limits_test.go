package p2p

import (
	"bufio"
	"bytes"
	"errors"
	"testing"
)

func TestReadBoundedPeerMessageAcceptsNormalFrame(t *testing.T) {
	want := []byte("{\"type\":\"ping\",\"payload\":{\"nonce\":1}}\n")
	got, err := readBoundedPeerMessage(bufio.NewReader(bytes.NewReader(want)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected frame: got %q want %q", got, want)
	}
}

func TestReadBoundedPeerMessageRejectsOversizedFrame(t *testing.T) {
	payload := bytes.Repeat([]byte{'x'}, MaxPeerMessageBytes+1)
	reader := bufio.NewReaderSize(bytes.NewReader(payload), 4096)

	_, err := readBoundedPeerMessage(reader)
	if !errors.Is(err, ErrPeerMessageTooLarge) {
		t.Fatalf("expected ErrPeerMessageTooLarge, got %v", err)
	}
}

func TestReadBoundedPeerMessageAllowsExactLimitWithNewline(t *testing.T) {
	payload := bytes.Repeat([]byte{'x'}, MaxPeerMessageBytes-1)
	payload = append(payload, '\n')

	got, err := readBoundedPeerMessage(bufio.NewReaderSize(bytes.NewReader(payload), 4096))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxPeerMessageBytes {
		t.Fatalf("expected %d bytes, got %d", MaxPeerMessageBytes, len(got))
	}
}
