package gpuminer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPoolFileConfig(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "deployment", "testnet", "gpu-miner-pool.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPoolFileConfig(filepath.Join("..", "deployment", "testnet", "gpu-miner-pool.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = raw
	if cfg.StratumURL == "" || cfg.RewardAddress == "" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Network() != "public-testnet" {
		t.Fatalf("network = %q", cfg.Network())
	}
}

func TestLoadPoolFileConfigRejectsMainnetUntilAuthorized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mainnet-pool.json")
	content := []byte(`{
	  "environment": "mainnet",
	  "stratum_url": "stratum+tcp://127.0.0.1:3333",
	  "reward_address": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPoolFileConfig(path); err == nil {
		t.Fatal("expected mainnet pool config rejection")
	}
}
