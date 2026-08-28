package demandminer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Environment: "public-testnet", StatusURL: "http://127.0.0.1:28545",
		ExpectedNetwork: "sudharma", ExpectedCoin: "Sudharma", ExpectedSymbol: "SUDH",
		SeedAddress: "127.0.0.1:28444", RewardAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
		MinerBinary: "/usr/local/bin/sudharmad", DataDirectory: "/var/lib/sudharma-demand-miner",
		LockFile: "/run/sudharma-demand-miner.lock", PollEvery: "10s", Cooldown: "30s",
		FailureBackoff: "30s", ChildTimeout: "5m",
	}
}

func TestConfigValidateAcceptsPublicTestnet(t *testing.T) {
	if err := validConfig().Validate(); err != nil { t.Fatalf("Validate: %v", err) }
}

func TestConfigRejectsMissingFields(t *testing.T) {
	cfg := validConfig(); cfg.RewardAddress = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "reward_address") { t.Fatalf("expected missing reward_address rejection, got %v", err) }
}

func TestConfigRejectsPublicStatusURL(t *testing.T) {
	cfg := validConfig(); cfg.StatusURL = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") { t.Fatalf("expected loopback rejection, got %v", err) }
}

func TestConfigRejectsMainnetIdentity(t *testing.T) {
	cfg := validConfig(); cfg.ExpectedCoin = "Sudharma Mainnet"
	if err := cfg.Validate(); err == nil { t.Fatal("expected testnet-only rejection") }
}

func TestConfigRejectsInvalidRewardAddress(t *testing.T) {
	cfg := validConfig(); cfg.RewardAddress = strings.Repeat("g", 40)
	if err := cfg.Validate(); err == nil { t.Fatal("expected invalid reward address rejection") }
	cfg.RewardAddress = strings.Repeat("a", 39)
	if err := cfg.Validate(); err == nil { t.Fatal("expected reward address length rejection") }
}

func TestConfigRejectsInvalidDurations(t *testing.T) {
	for _, field := range []string{"PollEvery", "Cooldown", "FailureBackoff"} {
		cfg := validConfig()
		switch field { case "PollEvery": cfg.PollEvery = "0s"; case "Cooldown": cfg.Cooldown = "-1s"; case "FailureBackoff": cfg.FailureBackoff = "nonsense" }
		if err := cfg.Validate(); err == nil { t.Fatalf("expected invalid %s rejection", field) }
	}
}

func TestConfigRejectsCooldownShorterThanPoll(t *testing.T) {
	cfg := validConfig(); cfg.PollEvery = "20s"; cfg.Cooldown = "10s"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "cooldown") { t.Fatalf("expected cooldown rejection, got %v", err) }
}

func TestConfigRejectsRelativePaths(t *testing.T) {
	cfg := validConfig(); cfg.MinerBinary = "./sudharmad"
	if err := cfg.Validate(); err == nil { t.Fatal("expected relative miner binary rejection") }
	cfg = validConfig(); cfg.DataDirectory = "data"
	if err := cfg.Validate(); err == nil { t.Fatal("expected relative data directory rejection") }
}

func TestConfigRejectsChildTimeoutBelowOneSecond(t *testing.T) {
	cfg := validConfig(); cfg.ChildTimeout = "999ms"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "child_timeout") { t.Fatalf("expected child timeout rejection, got %v", err) }
}

func TestLoadConfigRejectsUnknownFieldsAndParsesDurations(t *testing.T) {
	dir := t.TempDir(); path := filepath.Join(dir, "config.json")
	data := `{"environment":"public-testnet","status_url":"http://127.0.0.1:28545","expected_network":"sudharma","expected_coin":"Sudharma","expected_symbol":"SUDH","seed_address":"127.0.0.1:28444","reward_address":"9ccdc094489874bed888ffe4bdf9b8298f4c5131","miner_binary":"/bin/sudharmad","data_directory":"/tmp/miner","lock_file":"/tmp/miner.lock","poll_every":"10s","cooldown":"30s","failure_backoff":"30s","child_timeout":"5m","extra":true}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil { t.Fatal(err) }
	if _, err := LoadConfig(path); err == nil { t.Fatal("expected unknown field rejection") }

	cfg := validConfig(); if err := cfg.Validate(); err != nil { t.Fatal(err) }
	if got, err := cfg.PollDuration(); err != nil || got != 10*time.Second { t.Fatalf("PollDuration = %v, %v", got, err) }
}
