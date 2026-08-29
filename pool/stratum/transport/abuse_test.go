package transport

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestServeConnRateLimitStopsBeforeSecondRequest(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, newTransportFactory(t, &calls), Config{RequestsPerSecond: 1, Burst: 1})
	}()
	writeDone := make(chan error, 1)
	go func() {
		_, err := io.WriteString(client, "{\"id\":1,\"method\":\"mining.subscribe\",\"params\":[]}\n"+
			"{\"id\":2,\"method\":\"mining.subscribe\",\"params\":[]}\n")
		writeDone <- err
	}()
	first := readTransportJSONLine(t, bufio.NewReader(client))
	if id := int(first["id"].(float64)); id != 1 {
		t.Fatalf("handled response id = %d, want 1", id)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrRateLimited) {
			t.Fatalf("ServeConn error = %v, want ErrRateLimited", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not enforce rate limit")
	}
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("client write: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("client write did not finish")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}
}

func TestServeConnProtocolBudgetFramesTerminalError(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, newTransportFactory(t, &calls), Config{MaxProtocolErrors: 2})
	}()
	reader := bufio.NewReader(client)
	for i := 0; i < 3; i++ {
		if _, err := io.WriteString(client, "{\n"); err != nil {
			t.Fatal(err)
		}
		response := readTransportJSONLine(t, reader)
		protocolErr := response["error"].(map[string]any)
		if code := int(protocolErr["code"].(float64)); code != -32700 {
			t.Fatalf("protocol code = %d, want -32700", code)
		}
		if i < 2 {
			select {
			case err := <-done:
				t.Fatalf("connection ended before budget exceeded: %v", err)
			default:
			}
		}
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrProtocolBudget) {
			t.Fatalf("ServeConn error = %v, want ErrProtocolBudget", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not enforce protocol budget")
	}
	_ = client.Close()
}

func TestServeConnOversizedFragmentedLine(t *testing.T) {
	server, client := net.Pipe()
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- ServeConn(context.Background(), server, newTransportFactory(t, &calls), Config{})
	}()
	writeDone := make(chan error, 1)
	go func() {
		for _, fragment := range []string{strings.Repeat("x", 32*1024), strings.Repeat("x", 32*1024), "x\n"} {
			if _, err := io.WriteString(client, fragment); err != nil {
				writeDone <- err
				return
			}
		}
		writeDone <- nil
	}()
	response := readTransportJSONLine(t, bufio.NewReader(client))
	protocolErr := response["error"].(map[string]any)
	if code := int(protocolErr["code"].(float64)); code != -32600 {
		t.Fatalf("protocol code = %d, want -32600", code)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrLineTooLong) {
			t.Fatalf("ServeConn error = %v, want ErrLineTooLong", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ServeConn did not reject oversized line")
	}
	select {
	case err := <-writeDone:
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("client write: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("fragmented write did not finish")
	}
	_ = client.Close()
}
