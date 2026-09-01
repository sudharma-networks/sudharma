package pool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExamplePoolConfig(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "deployment", "testnet", "pool.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PayoutScheme != SchemePPLNS {
		t.Fatalf("scheme = %q", cfg.PayoutScheme)
	}
	if cfg.StratumListen != ":3333" {
		t.Fatalf("listen = %q", cfg.StratumListen)
	}
}

func TestResolveConfigRequiresPayoutAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RPCURL = "http://127.0.0.1:1"
	if _, err := ResolveConfig(cfg); err == nil {
		t.Fatal("expected payout address error")
	}
}
