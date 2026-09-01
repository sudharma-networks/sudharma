package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sudharma-networks/sudharma/pool"
)

func TestLoadExamplePoolConfigFromDeployment(t *testing.T) {
	path := filepath.Join("..", "..", "deployment", "testnet", "pool.example.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := pool.LoadConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PayoutScheme != pool.SchemePPLNS {
		t.Fatalf("scheme = %q", cfg.PayoutScheme)
	}
	if cfg.StratumListen != ":3333" {
		t.Fatalf("listen = %q", cfg.StratumListen)
	}
}

func TestResolvePoolConfigRequiresRPC(t *testing.T) {
	cfg := pool.DefaultConfig()
	cfg.PayoutAddress = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := pool.ResolveConfig(cfg); err == nil {
		t.Fatal("expected rpc requirement error")
	}
}

func TestProbeModeUsesEngine(t *testing.T) {
	if err := run([]string{
		"-network", "public-testnet",
		"-rpc", "http://127.0.0.1:1",
		"-payout-address", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"-probe",
	}); err == nil {
		t.Fatal("expected probe failure against unreachable RPC")
	}
}
