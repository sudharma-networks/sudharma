package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sudharma-networks/sudharma/params"
)

// Config contains operator-facing settings for a production Sudharma node.
// Secrets do not belong in this file; the current node RPC is read/query plus
// signed-transaction submission and never needs wallet private keys.
type Config struct {
	NodeID                string   `json:"node_id"`
	P2PAddress            string   `json:"p2p_address"`
	RPCAddress            string   `json:"rpc_address"`
	Peers                 []string `json:"peers"`
	DataDirectory         string   `json:"data_directory"`
	LogJSON               bool     `json:"log_json"`
	Metrics               bool     `json:"metrics"`
	PersistEvery          string   `json:"persist_every"`
	GPUV1ActivationHeight uint64   `json:"gpu_v1_activation_height"`
}

func DefaultConfig() Config {
	return Config{
		NodeID:                "rpc-node",
		P2PAddress:            "127.0.0.1:18444",
		RPCAddress:            "127.0.0.1:18545",
		DataDirectory:         "data-rpc-node",
		Metrics:               true,
		PersistEvery:          "30s",
		GPUV1ActivationHeight: params.GPUV1ActivationDisabled,
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}
	if c.P2PAddress == "" || c.RPCAddress == "" {
		return fmt.Errorf("p2p_address and rpc_address are required")
	}
	if c.DataDirectory == "" {
		return fmt.Errorf("data_directory is required")
	}
	if c.PersistEvery != "" {
		d, err := time.ParseDuration(c.PersistEvery)
		if err != nil || d < time.Second {
			return fmt.Errorf("persist_every must be a duration of at least 1s")
		}
	}
	return nil
}

func (c Config) PersistenceInterval() time.Duration {
	if c.PersistEvery == "" {
		return 0
	}
	d, _ := time.ParseDuration(c.PersistEvery)
	return d
}
