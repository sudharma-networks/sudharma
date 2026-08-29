package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

func TestServerReleasesAdmissionAfterNormalEOF(t *testing.T) {
	listener := newScriptedListener(2)
	remote := &net.TCPAddr{IP: net.ParseIP("198.51.100.50"), Port: 1000}
	serverOne, clientOne := net.Pipe()
	listener.push(&addressedConn{Conn: serverOne, remote: remote}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(ctx, listener, newServerFactory(&calls), transport.Config{}, Config{MaxConnections: 1, MaxConnectionsPerIP: 1})
	}()

	waitForServerCalls(t, &calls, 1)
	if err := clientOne.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	serverTwo, clientTwo := net.Pipe()
	defer clientTwo.Close()
	listener.push(&addressedConn{Conn: serverTwo, remote: &net.TCPAddr{IP: remote.IP, Port: 2000}}, nil)
	waitForServerCalls(t, &calls, 2)

	cancel()
	waitForServeListenerCanceled(t, done)
}

func TestServerReleasesAdmissionAfterRateLimitTermination(t *testing.T) {
	listener := newScriptedListener(2)
	remote := &net.TCPAddr{IP: net.ParseIP("198.51.100.51"), Port: 1000}
	serverOne, clientOne := net.Pipe()
	listener.push(&addressedConn{Conn: serverOne, remote: remote}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(
			ctx,
			listener,
			newServerFactory(&calls),
			transport.Config{RequestsPerSecond: 1, Burst: 1},
			Config{MaxConnections: 1, MaxConnectionsPerIP: 1},
		)
	}()

	waitForServerCalls(t, &calls, 1)
	reader := bufio.NewReader(clientOne)
	if _, err := io.WriteString(clientOne, `{"id":1,"method":"mining.subscribe","params":[]}`+"\n"); err != nil {
		t.Fatal(err)
	}
	_ = readServerJSONLine(t, reader)
	if _, err := io.WriteString(clientOne, `{"id":2,"method":"mining.subscribe","params":[]}`+"\n"); err != nil {
		t.Fatal(err)
	}
	assertConnectionClosed(t, clientOne, "rate-limited connection remained open")
	_ = clientOne.Close()
	time.Sleep(10 * time.Millisecond)

	serverTwo, clientTwo := net.Pipe()
	defer clientTwo.Close()
	listener.push(&addressedConn{Conn: serverTwo, remote: &net.TCPAddr{IP: remote.IP, Port: 2000}}, nil)
	waitForServerCalls(t, &calls, 2)

	cancel()
	waitForServeListenerCanceled(t, done)
}

func TestServerReleasesAdmissionAfterTLSFailure(t *testing.T) {
	listener := newScriptedListener(2)
	remoteIP := net.ParseIP("198.51.100.52")
	serverOne, clientOne := net.Pipe()
	listener.push(&addressedConn{Conn: serverOne, remote: &net.TCPAddr{IP: remoteIP, Port: 1000}}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(
			ctx,
			listener,
			newServerFactory(&calls),
			transport.Config{},
			Config{
				MaxConnections:      1,
				MaxConnectionsPerIP: 1,
				TLSConfig:           newTestServerTLSConfig(t),
				TLSHandshakeTimeout: time.Second,
			},
		)
	}()

	if _, err := io.WriteString(clientOne, "not tls\n"); err != nil {
		t.Fatal(err)
	}
	assertConnectionClosed(t, clientOne, "failed TLS connection remained open")
	_ = clientOne.Close()
	time.Sleep(10 * time.Millisecond)

	serverTwo, clientTwoRaw := net.Pipe()
	listener.push(&addressedConn{Conn: serverTwo, remote: &net.TCPAddr{IP: remoteIP, Port: 2000}}, nil)
	clientTwo := tls.Client(clientTwoRaw, newTestClientTLSConfig())
	defer clientTwo.Close()
	if err := clientTwo.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := clientTwo.Handshake(); err != nil {
		t.Fatalf("second TLS handshake after released admission: %v", err)
	}
	if err := clientTwo.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	waitForServerCalls(t, &calls, 1)

	cancel()
	waitForServeListenerCanceled(t, done)
}

func waitForServeListenerCanceled(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after cancellation")
	}
}
