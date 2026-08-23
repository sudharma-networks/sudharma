package p2p

import (
	"net"
	"testing"
	"time"
)

func TestSetPeerReadDeadlineRejectsNilConnection(t *testing.T) {
	if err := setPeerReadDeadline(nil); err == nil {
		t.Fatal("expected nil connection to be rejected")
	}
}

func TestSetPeerWriteDeadlineRejectsNilConnection(t *testing.T) {
	if err := setPeerWriteDeadline(nil); err == nil {
		t.Fatal("expected nil connection to be rejected")
	}
}

func TestPeerReadDeadlineExpiresStalledRead(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	if err := left.SetReadDeadline(time.Now().Add(25 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	buffer := make([]byte, 1)
	started := time.Now()
	_, err := left.Read(buffer)
	if err == nil {
		t.Fatal("expected stalled read to time out")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("stalled read took unexpectedly long to time out")
	}
}

func TestClearPeerDeadlinesAllowsTrafficAgain(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	if err := left.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := clearPeerDeadlines(left); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := right.Write([]byte{'x'})
		done <- err
	}()

	buffer := make([]byte, 1)
	if _, err := left.Read(buffer); err != nil {
		t.Fatalf("read failed after clearing deadline: %v", err)
	}
	if buffer[0] != 'x' {
		t.Fatalf("unexpected byte %q", buffer[0])
	}
	if err := <-done; err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
