package pool

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const DefaultPoolDifficulty = uint32(1)

// Config is the operator-facing pool configuration shape.
type Config struct {
	Network         string       `json:"network"`
	RPCURL          string       `json:"rpc_url"`
	RPCURLs         []string     `json:"rpc_urls,omitempty"`
	PayoutAddress   string       `json:"payout_address"`
	PayoutScheme    PayoutScheme `json:"payout_scheme"`
	PoolDifficulty  uint32       `json:"pool_difficulty"`
	PoolFeeBPS      uint64       `json:"pool_fee_bps"`
	PPLNSWindow     int          `json:"pplns_window"`
	StratumListen   string       `json:"stratum_listen"`
	WorkPollSeconds int          `json:"work_poll_seconds"`
	ExpectedNetwork string       `json:"expected_network,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Network:         "public-testnet",
		PayoutScheme:    SchemePPLNS,
		PoolDifficulty:  DefaultPoolDifficulty,
		PoolFeeBPS:      100,
		PPLNSWindow:     10_000,
		StratumListen:   ":3333",
		WorkPollSeconds: 5,
		ExpectedNetwork: "sudharma",
	}
}

func LoadConfig(raw []byte) (Config, error) {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return Config{}, fmt.Errorf("pool config is empty")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid pool config JSON: %w", err)
	}
	return ResolveConfig(cfg)
}

func ResolveConfig(cfg Config) (Config, error) {
	if strings.TrimSpace(cfg.PayoutAddress) == "" {
		return Config{}, fmt.Errorf("payout_address is required")
	}
	identity, err := ParseWorkerIdentity(cfg.PayoutAddress)
	if err != nil {
		return Config{}, fmt.Errorf("payout_address: %w", err)
	}
	cfg.PayoutAddress = identity.Address

	scheme, err := NormalizePayoutScheme(string(cfg.PayoutScheme))
	if err != nil {
		return Config{}, err
	}
	cfg.PayoutScheme = scheme

	if cfg.PoolDifficulty == 0 {
		cfg.PoolDifficulty = DefaultPoolDifficulty
	}
	if cfg.PPLNSWindow <= 0 {
		cfg.PPLNSWindow = 10_000
	}
	if strings.TrimSpace(cfg.StratumListen) == "" {
		cfg.StratumListen = ":3333"
	}
	if cfg.WorkPollSeconds <= 0 {
		cfg.WorkPollSeconds = 5
	}
	if strings.TrimSpace(cfg.ExpectedNetwork) == "" {
		cfg.ExpectedNetwork = "sudharma"
	}
	if strings.TrimSpace(cfg.Network) == "" {
		cfg.Network = "public-testnet"
	}
	if strings.TrimSpace(cfg.RPCURL) == "" && len(cfg.RPCURLs) == 0 {
		return Config{}, fmt.Errorf("rpc_url or rpc_urls is required")
	}
	return cfg, nil
}

func (c Config) WorkPollInterval() time.Duration {
	if c.WorkPollSeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.WorkPollSeconds) * time.Second
}
