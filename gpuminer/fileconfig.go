package gpuminer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/sudharma-networks/sudharma/params"
)

// FileConfig mirrors the demand-miner deployment shape for public-testnet GPU
// mining. Public clients use public_rpc_url; operators on the seed VPC may
// reach seed1_rpc_url and seed2_rpc_url directly.
type FileConfig struct {
	Environment     string `json:"environment"`
	PublicRPCURL    string `json:"public_rpc_url"`
	Seed1RPCURL     string `json:"seed1_rpc_url"`
	Seed2RPCURL     string `json:"seed2_rpc_url"`
	ExpectedNetwork string `json:"expected_network"`
	ExpectedCoin    string `json:"expected_coin"`
	ExpectedSymbol  string `json:"expected_symbol"`
	RewardAddress   string `json:"reward_address"`
	Backend         string `json:"backend"`
	Device          int    `json:"device"`
}

func LoadFileConfig(path string) (FileConfig, error) {
	var cfg FileConfig
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

func (c FileConfig) Validate() error {
	if c.Environment == "" {
		return errors.New("environment is required")
	}
	if c.Environment != params.NetworkPublicTestnet {
		return errors.New("environment must be public-testnet")
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
	if !rewardAddressPattern.MatchString(c.RewardAddress) {
		return errors.New("reward_address must be 40 lowercase hexadecimal characters")
	}
	if err := validateMiningRPCURL(c.PublicRPCURL); err != nil {
		return fmt.Errorf("public_rpc_url: %w", err)
	}
	if err := validateMiningRPCURL(c.Seed1RPCURL); err != nil {
		return fmt.Errorf("seed1_rpc_url: %w", err)
	}
	if err := validateMiningRPCURL(c.Seed2RPCURL); err != nil {
		return fmt.Errorf("seed2_rpc_url: %w", err)
	}
	if c.Backend != "" {
		if err := params.ValidateMiningBackend(c.Backend); err != nil {
			return fmt.Errorf("backend: %w", err)
		}
	}
	return nil
}

func (c FileConfig) ToConfig() (Config, error) {
	address, err := NormalizeRewardAddress(c.RewardAddress)
	if err != nil {
		return Config{}, err
	}
	backend := strings.TrimSpace(c.Backend)
	if backend == "" {
		backend = params.ProductionMiningBackend
	}
	return Config{
		Address:         address,
		Network:         params.NetworkPublicTestnet,
		RPCURLs:         c.MiningEndpoints(),
		Backend:         backend,
		Device:          c.Device,
		ExpectedNetwork: c.ExpectedNetwork,
		ExpectedCoin:    c.ExpectedCoin,
		ExpectedSymbol:  c.ExpectedSymbol,
	}, nil
}

func (c FileConfig) MiningEndpoints() []string {
	seen := make(map[string]struct{}, 3)
	var endpoints []string
	add := func(raw string) {
		trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		endpoints = append(endpoints, trimmed)
	}
	add(c.Seed1RPCURL)
	add(c.Seed2RPCURL)
	add(c.PublicRPCURL)
	if len(endpoints) == 0 {
		if defaults, err := params.MiningRPCEndpointsForNetwork(params.NetworkPublicTestnet); err == nil {
			return defaults
		}
	}
	return endpoints
}

func validateMiningRPCURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("must be an absolute http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("scheme must be http or https")
	}
	return nil
}
