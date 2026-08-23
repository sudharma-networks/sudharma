package operations

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaultsAndOverrides(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NodeID != "rpc-node" || cfg.PersistenceInterval() != 30*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	path := filepath.Join(t.TempDir(), "node.json")
	if err := os.WriteFile(path, []byte(`{"node_id":"public-1","p2p_address":"0.0.0.0:18444","rpc_address":"127.0.0.1:18545","data_directory":"node-data","peers":["127.0.0.1:18445"],"persist_every":"5s"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NodeID != "public-1" || cfg.PersistenceInterval() != 5*time.Second || len(cfg.Peers) != 1 {
		t.Fatalf("override not applied: %+v", cfg)
	}
}

func TestConfigRejectsUnsafeInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PersistEvery = "100ms"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected persistence interval validation error")
	}
	cfg = DefaultConfig()
	cfg.DataDirectory = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected data directory validation error")
	}
}
