package demandminer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{
		Environment:     "public-testnet",
		StatusURL:       "http://127.0.0.1:28545",
		ExpectedNetwork: "sudharma",
		ExpectedCoin:    "Sudharma",
		ExpectedSymbol:  "SUDH",
		SeedAddress:     "127.0.0.1:28444",
		RewardAddress:   "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
		MinerBinary:     "/usr/local/bin/sudharmad",
		DataDirectory:   "/var/lib/sudharma-demand-miner",
		LockFile:        "/run/sudharma-demand-miner.lock",
		PollEvery:       "10s",
		Cooldown:        "30s",
		FailureBackoff:  "30s",
		ChildTimeout:    "5m",
	}
}

func TestConfigValidateAcceptsPublicTestnet(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfigValidateRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"missing environment", func(c *Config) { c.Environment = "" }, "environment"},
		{"wrong environment", func(c *Config) { c.Environment = "mainnet" }, "public-testnet"},
		{"public status URL", func(c *Config) { c.StatusURL = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com" }, "loopback"},
		{"wrong network", func(c *Config) { c.ExpectedNetwork = "other" }, "network"},
		{"wrong coin", func(c *Config) { c.ExpectedCoin = "Sudharma Mainnet" }, "coin"},
		{"wrong symbol", func(c *Config) { c.ExpectedSymbol = "SDH" }, "symbol"},
		{"bad reward address", func(c *Config) { c.RewardAddress = "ABC" }, "reward"},
		{"relative miner binary", func(c *Config) { c.MinerBinary = "sudharmad" }, "miner_binary"},
		{"relative data directory", func(c *Config) { c.DataDirectory = "data" }, "data_directory"},
		{"relative lock file", func(c *Config) { c.LockFile = "miner.lock" }, "lock_file"},
		{"zero poll", func(c *Config) { c.PollEvery = "0s" }, "poll_every"},
		{"cooldown shorter than poll", func(c *Config) { c.Cooldown = "5s" }, "cooldown"},
		{"zero backoff", func(c *Config) { c.FailureBackoff = "0s" }, "failure_backoff"},
		{"short child timeout", func(c *Config) { c.ChildTimeout = "500ms" }, "child_timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{"environment":"public-testnet","status_url":"http://127.0.0.1:28545","expected_network":"sudharma","expected_coin":"Sudharma","expected_symbol":"SUDH","seed_address":"127.0.0.1:28444","reward_address":"9ccdc094489874bed888ffe4bdf9b8298f4c5131","miner_binary":"/usr/local/bin/sudharmad","data_directory":"/var/lib/sudharma-demand-miner","lock_file":"/run/sudharma-demand-miner.lock","poll_every":"10s","cooldown":"30s","failure_backoff":"30s","child_timeout":"5m","unexpected":true}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil || !strings.Contains(strings.ToLower(err.Error()), "unknown") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
}

func TestLoadConfigReturnsParsedDurations(t *testing.T) {
	cfg := validConfig()
	if cfg.PollDuration() != 10_000_000_000 {
		t.Fatalf("unexpected poll duration %v", cfg.PollDuration())
	}
	if cfg.CooldownDuration() != 30_000_000_000 {
		t.Fatalf("unexpected cooldown duration %v", cfg.CooldownDuration())
	}
	if cfg.FailureBackoffDuration() != 30_000_000_000 {
		t.Fatalf("unexpected backoff %v", cfg.FailureBackoffDuration())
	}
	if cfg.ChildTimeoutDuration() != 300_000_000_000 {
		t.Fatalf("unexpected timeout %v", cfg.ChildTimeoutDuration())
	}
}
