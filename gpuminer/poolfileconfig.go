package gpuminer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sudharma-networks/sudharma/params"
)

// PoolFileConfig connects a GPU miner to a Sudharma Stratum pool.
type PoolFileConfig struct {
	Environment   string `json:"environment"`
	StratumURL    string `json:"stratum_url"`
	RewardAddress string `json:"reward_address"`
	WorkerName    string `json:"worker_name"`
	Password      string `json:"password"`
	Backend       string `json:"backend"`
	Device        int    `json:"device"`
}

func LoadPoolFileConfig(path string) (PoolFileConfig, error) {
	var cfg PoolFileConfig
	f, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("open pool config: %w", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode pool config: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return cfg, errors.New("decode pool config: multiple JSON values")
		}
		return cfg, fmt.Errorf("decode pool config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c PoolFileConfig) Validate() error {
	if c.Environment == "" {
		return errors.New("environment is required")
	}
	network := params.NormalizeMiningNetwork(c.Environment)
	if network != params.MiningNetworkPublicTestnet && network != params.MiningNetworkMainnet {
		return fmt.Errorf("environment must be %q or %q", params.MiningNetworkPublicTestnet, params.MiningNetworkMainnet)
	}
	if network == params.MiningNetworkMainnet && !params.MainnetMiningAuthorized {
		return fmt.Errorf("mainnet pool mining is not authorized until launch")
	}
	if strings.TrimSpace(c.StratumURL) == "" {
		return errors.New("stratum_url is required")
	}
	if !rewardAddressPattern.MatchString(c.RewardAddress) {
		return errors.New("reward_address must be 40 lowercase hexadecimal characters")
	}
	if c.Backend != "" {
		if err := params.ValidateMiningBackend(c.Backend); err != nil {
			return fmt.Errorf("backend: %w", err)
		}
	}
	return nil
}

func (c PoolFileConfig) Network() string {
	return params.NormalizeMiningNetwork(c.Environment)
}
