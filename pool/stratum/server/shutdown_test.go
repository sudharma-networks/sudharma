package server

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

func TestServeListenerCancellationClosesBlockedListener(t *testing.T) {
	listener := newScriptedListener(0)
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(ctx, listener, newServerFactory(&calls), transport.Config{}, Config{})
	}()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not wake from blocked Accept on cancellation")
	}
	select {
	case <-listener.closed:
	default:
		t.Fatal("listener was not closed during cancellation")
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("session factory calls = %d, want 0", got)
	}
}

func TestServeListenerCancellationJoinsActiveConnections(t *testing.T) {
	listener := newScriptedListener(3)
	clients := make([]net.Conn, 0, 3)
	for i := 0; i < 3; i++ {
		serverConn, clientConn := net.Pipe()
		clients = append(clients, clientConn)
		listener.push(&addressedConn{
			Conn: serverConn,
			remote: &net.TCPAddr{
				IP:   net.ParseIP("198.51.100." + string(rune('1'+i))),
				Port: 8000 + i,
			},
		}, nil)
	}
	defer func() {
		for _, client := range clients {
			_ = client.Close()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(
			ctx,
			listener,
			newServerFactory(&calls),
			transport.Config{},
			Config{MaxConnections: 3, MaxConnectionsPerIP: 1},
		)
	}()

	waitForServerCalls(t, &calls, 3)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener returned before active connections were joined")
	}

	for _, client := range clients {
		assertConnectionClosed(t, client, "active connection remained open after ServeListener returned")
	}
}
