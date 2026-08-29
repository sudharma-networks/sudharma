package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestServeConnReadDeadline(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, newTransportFactory(t, &calls), Config{ReadTimeout: 20 * time.Millisecond})
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "read Stratum request") {
			t.Fatalf("ServeConn error = %v, want read timeout context", err)
		}
	case <-time.After(time.Second):
		t.Fatal("read deadline did not stop ServeConn")
	}
	_ = client.Close()
}

func TestServeConnWriteDeadline(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, newTransportFactory(t, &calls), Config{WriteTimeout: 20 * time.Millisecond})
	}()
	writeDone := make(chan error, 1)
	go func() {
		_, err := io.WriteString(client, "{\"id\":1,\"method\":\"mining.subscribe\",\"params\":[]}\n")
		writeDone <- err
	}()
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("client request write blocked")
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "write Stratum response") {
			t.Fatalf("ServeConn error = %v, want write timeout context", err)
		}
	case <-time.After(time.Second):
		t.Fatal("write deadline did not stop ServeConn")
	}
	_ = client.Close()
}

func TestServeConnCancelWakesBlockedRead(t *testing.T) {
	server, client := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(ctx, server, newTransportFactory(t, &calls), Config{})
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ServeConn error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not wake blocked read")
	}
	_ = client.Close()
}
