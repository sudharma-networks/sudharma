package server

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

type temporaryAcceptError struct{}

func (temporaryAcceptError) Error() string   { return "temporary accept failure" }
func (temporaryAcceptError) Timeout() bool   { return false }
func (temporaryAcceptError) Temporary() bool { return true }

func TestServeListenerRetriesTemporaryAcceptError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	listener := newScriptedListener(2)
	listener.push(nil, temporaryAcceptError{})
	listener.push(&addressedConn{
		Conn:   serverConn,
		remote: &net.TCPAddr{IP: net.ParseIP("192.0.2.30"), Port: 7000},
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(
			ctx,
			listener,
			newServerFactory(&calls),
			transport.Config{},
			Config{AcceptErrorBackoff: time.Millisecond},
		)
	}()

	waitForServerCalls(t, &calls, 1)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after temporary-error retry")
	}
}

func TestServeListenerPermanentAcceptErrorTerminates(t *testing.T) {
	want := errors.New("permanent accept failure")
	listener := newScriptedListener(1)
	listener.push(nil, want)

	var calls atomic.Int32
	err := ServeListener(
		context.Background(),
		listener,
		newServerFactory(&calls),
		transport.Config{},
		Config{},
	)
	if !errors.Is(err, want) {
		t.Fatalf("ServeListener error = %v, want wrapped permanent error", err)
	}
	if !strings.Contains(err.Error(), "accept Stratum connection") {
		t.Fatalf("ServeListener error = %q, want operation context", err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("session factory calls = %d, want 0", got)
	}
}
