package gpuminer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/params"
)

func TestLoadFileConfigMatchesDemandMinerShape(t *testing.T) {
	path := filepath.Join("..", "deployment", "testnet", "gpu-miner.example.json")
	cfg, err := LoadFileConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != params.MiningNetworkPublicTestnet {
		t.Fatalf("environment = %q", cfg.Environment)
	}
	if cfg.ExpectedNetwork != "sudharma" || cfg.ExpectedCoin != "Sudharma" || cfg.ExpectedSymbol != "SUDH" {
		t.Fatalf("identity = %+v", cfg)
	}
	endpoints := cfg.MiningEndpoints()
	if len(endpoints) != 3 {
		t.Fatalf("endpoints = %#v", endpoints)
	}
	if endpoints[0] != params.PublicTestnetSeed1RPC {
		t.Fatalf("seed-1 first for deployment config = %q", endpoints[0])
	}
	if endpoints[1] != params.PublicTestnetSeed2RPC {
		t.Fatalf("seed-2 second = %q", endpoints[1])
	}
	if endpoints[2] != params.PublicTestnetMiningRPC {
		t.Fatalf("public proxy last = %q", endpoints[2])
	}
}

func TestFailoverClientUsesSeed2WhenSeed1Unavailable(t *testing.T) {
	var calls []string
	seed1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "seed1:"+r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"seed unavailable"}`))
	}))
	t.Cleanup(seed1.Close)

	seed2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "seed2:"+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NetworkStatus{Network: "sudharma", Height: 42})
	}))
	t.Cleanup(seed2.Close)

	client, err := NewFailoverClient([]string{seed1.URL, seed2.URL}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	status, err := client.NetworkStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Height != 42 {
		t.Fatalf("height = %d", status.Height)
	}
	if len(calls) != 2 || !strings.HasPrefix(calls[0], "seed1:") || !strings.HasPrefix(calls[1], "seed2:") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestValidateNetworkStatus(t *testing.T) {
	if err := ValidateNetworkStatus(NetworkStatus{Network: "sudharma"}, "sudharma"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNetworkStatus(NetworkStatus{Network: "other"}, "sudharma"); err == nil {
		t.Fatal("expected mismatch")
	}
}
