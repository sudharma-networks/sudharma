package rpc

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestClientQueriesServer(t *testing.T) {
	server, _, _, state := newTestServer(t)
	if err := state.Credit("client-account", 42); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client, err := NewClient(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.NodeID != "rpc-test-node" {
		t.Fatalf("unexpected node ID: %s", status.NodeID)
	}
	if status.GPUV1.Phase != "disabled" {
		t.Fatalf("unexpected GPU v1 phase: %q", status.GPUV1.Phase)
	}
	if status.GPUV1.ActivationHeight != nil {
		t.Fatalf("disabled GPU v1 status exposed activation height: %d", *status.GPUV1.ActivationHeight)
	}
	if status.GPUV1.NextBlockVersion != 1 {
		t.Fatalf("unexpected next block version: %d", status.GPUV1.NextBlockVersion)
	}
	account, err := client.Account(context.Background(), "client-account")
	if err != nil {
		t.Fatal(err)
	}
	if account.Balance != 42 || account.NextNonce != 1 {
		t.Fatalf("unexpected account: %+v", account)
	}
}

func TestClientReturnsStructuredRPCError(t *testing.T) {
	server, _, _, _ := newTestServer(t)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	client, err := NewClient(httpServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Transaction(context.Background(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err == nil {
		t.Fatal("expected missing transaction error")
	}
	rpcErr, ok := err.(*RPCError)
	if !ok || rpcErr.StatusCode != 404 {
		t.Fatalf("unexpected RPC error: %T %v", err, err)
	}
}

func TestNewClientRejectsUnsafeURLShapes(t *testing.T) {
	for _, value := range []string{"ftp://127.0.0.1", "not-a-url"} {
		if _, err := NewClient(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
