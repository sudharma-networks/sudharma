package server

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pool/stratum/transport"
)

func TestServeListenerRejectsInvalidArguments(t *testing.T) {
	var calls atomic.Int32
	factory := newServerFactory(&calls)
	if err := ServeListener(context.Background(), nil, factory, transport.Config{}, Config{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("nil listener error = %v, want ErrInvalidConfig", err)
	}

	listener := newScriptedListener(0)
	if err := ServeListener(context.Background(), listener, nil, transport.Config{}, Config{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("nil factory error = %v, want ErrInvalidConfig", err)
	}
}

func TestServeListenerGlobalCapRejectsBeforeSessionCreation(t *testing.T) {
	serverOne, clientOne := net.Pipe()
	serverTwo, clientTwo := net.Pipe()
	defer clientOne.Close()
	defer clientTwo.Close()

	listener := newScriptedListener(2)
	listener.push(&addressedConn{Conn: serverOne, remote: &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 1000}}, nil)
	listener.push(&addressedConn{Conn: serverTwo, remote: &net.TCPAddr{IP: net.ParseIP("192.0.2.11"), Port: 2000}}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(ctx, listener, newServerFactory(&calls), transport.Config{}, Config{MaxConnections: 1, MaxConnectionsPerIP: 1})
	}()

	waitForServerCalls(t, &calls, 1)
	if err := clientTwo.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var one [1]byte
	if _, err := clientTwo.Read(one[:]); err == nil {
		t.Fatal("globally rejected connection remained open")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after cancellation")
	}
}

func TestServeListenerPerIPCapAllowsDifferentSource(t *testing.T) {
	serverOne, clientOne := net.Pipe()
	serverTwo, clientTwo := net.Pipe()
	serverThree, clientThree := net.Pipe()
	defer clientOne.Close()
	defer clientTwo.Close()
	defer clientThree.Close()

	listener := newScriptedListener(3)
	ipOne := net.ParseIP("198.51.100.10")
	listener.push(&addressedConn{Conn: serverOne, remote: &net.TCPAddr{IP: ipOne, Port: 1000}}, nil)
	listener.push(&addressedConn{Conn: serverTwo, remote: &net.TCPAddr{IP: ipOne, Port: 2000}}, nil)
	listener.push(&addressedConn{Conn: serverThree, remote: &net.TCPAddr{IP: net.ParseIP("198.51.100.11"), Port: 3000}}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(ctx, listener, newServerFactory(&calls), transport.Config{}, Config{MaxConnections: 2, MaxConnectionsPerIP: 1})
	}()

	waitForServerCalls(t, &calls, 2)
	if err := clientTwo.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var one [1]byte
	if _, err := clientTwo.Read(one[:]); err == nil {
		t.Fatal("per-IP rejected connection remained open")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("session factory calls = %d, want 2", got)
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after cancellation")
	}
}

func TestServeListenerPlaintextDelegatesToStageE(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	listener := newScriptedListener(1)
	listener.push(&addressedConn{Conn: serverConn, remote: &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 4040}}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeListener(ctx, listener, newServerFactory(&calls), transport.Config{}, Config{})
	}()

	if _, err := io.WriteString(clientConn, `{"id":1,"method":"mining.subscribe","params":[]}`+"\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(clientConn)
	subscribe := readServerJSONLine(t, reader)
	if id := int(subscribe["id"].(float64)); id != 1 {
		t.Fatalf("subscribe response id = %d, want 1", id)
	}

	authorize := `{"id":2,"method":"mining.authorize","params":["` + serverWallet + `.rig_01","x"]}` + "\n"
	if _, err := io.WriteString(clientConn, authorize); err != nil {
		t.Fatal(err)
	}
	response := readServerJSONLine(t, reader)
	if response["result"] != true {
		t.Fatalf("authorize result = %v, want true", response["result"])
	}
	if got := readServerJSONLine(t, reader)["method"]; got != "mining.set_difficulty" {
		t.Fatalf("first work method = %v", got)
	}
	if got := readServerJSONLine(t, reader)["method"]; got != "mining.notify" {
		t.Fatalf("second work method = %v", got)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeListener error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeListener did not stop after cancellation")
	}
}
