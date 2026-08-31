package demandminer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var rewardAddressPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type Config struct {
	Environment         string `json:"environment"`
	StatusURL           string `json:"status_url"`
	ExpectedNetwork     string `json:"expected_network"`
	ExpectedCoin        string `json:"expected_coin"`
	ExpectedSymbol      string `json:"expected_symbol"`
	SeedAddress         string `json:"seed_address"`
	RewardAddress       string `json:"reward_address"`
	MinerBinary         string `json:"miner_binary"`
	DataDirectory       string `json:"data_directory"`
	LockFile            string `json:"lock_file"`
	PollEvery           string `json:"poll_every"`
	Cooldown            string `json:"cooldown"`
	FailureBackoff      string `json:"failure_backoff"`
	ChildTimeout        string `json:"child_timeout"`
	ScheduledSweepEvery string `json:"scheduled_sweep_every"`
	MaxBlocksPerSweep   int    `json:"max_blocks_per_sweep"`
	FaucetMinBalance    uint64 `json:"faucet_min_balance"`
	FaucetFundingBlocks int    `json:"faucet_funding_blocks"`
	WakeListen          string `json:"wake_listen"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return cfg, errors.New("decode config: multiple JSON values")
		}
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Environment == "" {
		return errors.New("environment is required")
	}
	if c.Environment != "public-testnet" {
		return errors.New("environment must be public-testnet")
	}
	if err := validateLoopbackURL(c.StatusURL); err != nil {
		return fmt.Errorf("status_url: %w", err)
	}
	if c.ExpectedNetwork != "sudharma" {
		return errors.New("expected_network must be sudharma")
	}
	if c.ExpectedCoin != "Sudharma" {
		return errors.New("expected_coin must be Sudharma")
	}
	if c.ExpectedSymbol != "SUDH" {
		return errors.New("expected_symbol must be SUDH")
	}
	if err := validateHostPort(c.SeedAddress); err != nil {
		return fmt.Errorf("seed_address: %w", err)
	}
	if !rewardAddressPattern.MatchString(c.RewardAddress) {
		return errors.New("reward_address must be 40 lowercase hexadecimal characters")
	}
	if !filepath.IsAbs(c.MinerBinary) {
		return errors.New("miner_binary must be an absolute path")
	}
	if !filepath.IsAbs(c.DataDirectory) {
		return errors.New("data_directory must be an absolute path")
	}
	if !filepath.IsAbs(c.LockFile) {
		return errors.New("lock_file must be an absolute path")
	}

	poll, err := positiveDuration("poll_every", c.PollEvery)
	if err != nil {
		return err
	}
	cooldown, err := positiveDuration("cooldown", c.Cooldown)
	if err != nil {
		return err
	}
	if cooldown < poll {
		return errors.New("cooldown must not be shorter than poll_every")
	}
	if _, err := positiveDuration("failure_backoff", c.FailureBackoff); err != nil {
		return err
	}
	child, err := positiveDuration("child_timeout", c.ChildTimeout)
	if err != nil {
		return err
	}
	if child < time.Second {
		return errors.New("child_timeout must be at least 1s")
	}
	if c.ScheduledSweepEvery != "" {
		if _, err := positiveDuration("scheduled_sweep_every", c.ScheduledSweepEvery); err != nil {
			return err
		}
	}
	if c.MaxBlocksPerSweep < 0 {
		return errors.New("max_blocks_per_sweep must not be negative")
	}
	if c.WakeListen != "" {
		if err := validateLoopbackHostPort(c.WakeListen); err != nil {
			return fmt.Errorf("wake_listen: %w", err)
		}
	}
	return nil
}

func validateLoopbackURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return errors.New("is required")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("must be an absolute URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("scheme must be http or https")
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("host must be loopback")
	}
	return nil
}

func validateHostPort(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("is required")
	}
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" || port == "" {
		return errors.New("must be host:port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("port must be a number from 1 to 65535")
	}
	return nil
}

func positiveDuration(name, raw string) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", name, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return d, nil
}

func (c Config) PollDuration() time.Duration     { d, _ := time.ParseDuration(c.PollEvery); return d }
func (c Config) CooldownDuration() time.Duration { d, _ := time.ParseDuration(c.Cooldown); return d }
func (c Config) FailureBackoffDuration() time.Duration {
	d, _ := time.ParseDuration(c.FailureBackoff)
	return d
}
func (c Config) ChildTimeoutDuration() time.Duration {
	d, _ := time.ParseDuration(c.ChildTimeout)
	return d
}
func (c Config) ScheduledSweepDuration() time.Duration {
	if strings.TrimSpace(c.ScheduledSweepEvery) == "" {
		return 30 * time.Minute
	}
	d, _ := time.ParseDuration(c.ScheduledSweepEvery)
	return d
}
func (c Config) BlocksPerSweepLimit() int {
	if c.MaxBlocksPerSweep <= 0 {
		return 32
	}
	return c.MaxBlocksPerSweep
}

func (c Config) FaucetFundingBlocksLimit() int {
	if c.FaucetFundingBlocks <= 0 {
		return 2
	}
	return c.FaucetFundingBlocks
}

func (c Config) WakeListenAddress() string {
	if strings.TrimSpace(c.WakeListen) == "" {
		return DefaultWakeListen
	}
	return c.WakeListen
}
