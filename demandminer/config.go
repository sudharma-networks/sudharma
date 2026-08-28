// Package demandminer contains the public-testnet demand miner supervisor.
package demandminer

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config is the non-secret configuration for the demand miner. It deliberately
// contains no private keys or wallet credentials.
type Config struct {
	Environment     string `json:"environment"`
	StatusURL       string `json:"status_url"`
	ExpectedNetwork string `json:"expected_network"`
	ExpectedCoin    string `json:"expected_coin"`
	ExpectedSymbol  string `json:"expected_symbol"`
	SeedAddress     string `json:"seed_address"`
	RewardAddress   string `json:"reward_address"`
	MinerBinary     string `json:"miner_binary"`
	DataDirectory   string `json:"data_directory"`
	LockFile        string `json:"lock_file"`
	PollEvery       string `json:"poll_every"`
	Cooldown        string `json:"cooldown"`
	FailureBackoff  string `json:"failure_backoff"`
	ChildTimeout    string `json:"child_timeout"`
}

var rewardAddressPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// LoadConfig reads and strictly decodes a JSON configuration file.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	if strings.TrimSpace(path) == "" {
		return cfg, fmt.Errorf("configuration path is required")
	}
	f, err := os.Open(path)
	if err != nil { return cfg, fmt.Errorf("open config: %w", err) }
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil { return cfg, fmt.Errorf("decode config: %w", err) }
	var extra any
	if err := dec.Decode(&extra); err != io.EOF { return cfg, fmt.Errorf("config must contain one JSON object") }
	if err := cfg.Validate(); err != nil { return cfg, err }
	return cfg, nil
}

// Validate enforces the fixed public-testnet identity and safe local endpoints.
func (c Config) Validate() error {
	required := map[string]string{
		"environment": c.Environment, "status_url": c.StatusURL, "expected_network": c.ExpectedNetwork,
		"expected_coin": c.ExpectedCoin, "expected_symbol": c.ExpectedSymbol, "seed_address": c.SeedAddress,
		"reward_address": c.RewardAddress, "miner_binary": c.MinerBinary, "data_directory": c.DataDirectory,
		"lock_file": c.LockFile, "poll_every": c.PollEvery, "cooldown": c.Cooldown,
		"failure_backoff": c.FailureBackoff, "child_timeout": c.ChildTimeout,
	}
	for name, value := range required {
		if strings.TrimSpace(value) == "" { return fmt.Errorf("%s is required", name) }
	}
	if c.Environment != "public-testnet" { return fmt.Errorf("environment must be public-testnet") }
	if c.ExpectedNetwork != "sudharma" { return fmt.Errorf("expected_network must be sudharma") }
	if c.ExpectedCoin != "Sudharma" { return fmt.Errorf("expected_coin must be Sudharma") }
	if c.ExpectedSymbol != "SUDH" { return fmt.Errorf("expected_symbol must be SUDH") }

	u, err := url.Parse(c.StatusURL)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil {
		return fmt.Errorf("status_url must be a valid loopback URL")
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() { return fmt.Errorf("status_url must use a loopback address") }
	if u.Scheme != "http" && u.Scheme != "https" { return fmt.Errorf("status_url must use http or https") }
	if !rewardAddressPattern.MatchString(c.RewardAddress) { return fmt.Errorf("reward_address must be exactly 40 lowercase hexadecimal characters") }
	for name, path := range map[string]string{"miner_binary": c.MinerBinary, "data_directory": c.DataDirectory, "lock_file": c.LockFile} {
		if !filepath.IsAbs(path) { return fmt.Errorf("%s must be an absolute path", name) }
	}
	poll, err := c.PollDuration(); if err != nil { return err }
	cooldown, err := c.CooldownDuration(); if err != nil { return err }
	if _, err := c.FailureBackoffDuration(); err != nil { return err }
	if _, err := c.ChildTimeoutDuration(); err != nil { return err }
	if cooldown < poll { return fmt.Errorf("cooldown must be at least poll_every") }
	return nil
}

func parsePositiveDuration(name, value string, minimum time.Duration) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 { return 0, fmt.Errorf("%s must be a positive duration", name) }
	if d < minimum { return 0, fmt.Errorf("%s must be at least %s", name, minimum) }
	return d, nil
}

func (c Config) PollDuration() (time.Duration, error) { return parsePositiveDuration("poll_every", c.PollEvery, 0) }
func (c Config) CooldownDuration() (time.Duration, error) { return parsePositiveDuration("cooldown", c.Cooldown, 0) }
func (c Config) FailureBackoffDuration() (time.Duration, error) { return parsePositiveDuration("failure_backoff", c.FailureBackoff, 0) }
func (c Config) ChildTimeoutDuration() (time.Duration, error) { return parsePositiveDuration("child_timeout", c.ChildTimeout, time.Second) }
