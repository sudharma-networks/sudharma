package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/rpc"
)

type fakeStatusClient struct {
	status *rpc.NodeStatus
	err    error
}

func (f fakeStatusClient) Status(context.Context) (*rpc.NodeStatus, error) {
	return f.status, f.err
}

func TestRunRejectsMissingConfig(t *testing.T) {
	var logs bytes.Buffer
	err := run(context.Background(), nil, &logs)
	if err == nil || !strings.Contains(err.Error(), "-config") {
		t.Fatalf("expected missing -config error, got %v", err)
	}
}

func TestAcquireLockRejectsSecondProcess(t *testing.T) {
	path := t.TempDir() + "/miner.lock"
	first, err := acquireLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	if _, err := acquireLock(first.Name()); err == nil {
		t.Fatal("expected lock contention")
	}
}

func TestRPCStatusSourceMapsAllSupervisorFields(t *testing.T) {
	source := rpcStatusSource{client: fakeStatusClient{status: &rpc.NodeStatus{
		Network:      "sudharma",
		Coin:         "Sudharma",
		Symbol:       "SUDH",
		Height:       42,
		Mempool:      3,
		IssuedSupply: 5000000000,
	}}}

	got, err := source.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Network != "sudharma" || got.Coin != "Sudharma" || got.Symbol != "SUDH" || got.Height != 42 || got.Mempool != 3 || got.IssuedSupply != 5000000000 {
		t.Fatalf("unexpected mapped status: %+v", got)
	}
}

func TestRPCStatusSourcePropagatesErrors(t *testing.T) {
	want := errors.New("rpc unavailable")
	source := rpcStatusSource{client: fakeStatusClient{err: want}}
	if _, err := source.Status(context.Background()); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestJSONLoggerEmitsEventWithoutConfigurationContents(t *testing.T) {
	var out bytes.Buffer
	logger := newJSONLogger(&out)
	logger.Error("mine_error", errors.New("child failed"))

	text := out.String()
	if !strings.Contains(text, `"event":"mine_error"`) || !strings.Contains(text, `"error":"child failed"`) {
		t.Fatalf("unexpected JSON log: %s", text)
	}
	for _, forbidden := range []string{"reward_address", "miner_binary", "data_directory", "9ccdc094489874bed888ffe4bdf9b8298f4c5131"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("log leaked configuration field %q: %s", forbidden, text)
		}
	}
}
