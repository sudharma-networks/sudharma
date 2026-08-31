package gpuminer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sudharma-networks/sudharma/params"
)

var rewardAddressPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type Config struct {
	Address string
	Network string
	RPCURL  string
	Backend string
	Device  int
}

type Work struct {
	WorkID        string `json:"work_id"`
	Algorithm     string `json:"algorithm"`
	Version       uint32 `json:"version"`
	Height        uint64 `json:"height"`
	Difficulty    uint32 `json:"difficulty"`
	Target        string `json:"target"`
	HeaderPrefix  string `json:"header_prefix"`
	RewardAddress string `json:"reward_address"`
}

type Solution struct {
	WorkID        string `json:"work_id"`
	Nonce         uint64 `json:"nonce"`
	Algorithm     string `json:"algorithm"`
	Version       uint32 `json:"version"`
	Height        uint64 `json:"height"`
	Difficulty    uint32 `json:"difficulty"`
	Target        string `json:"target"`
	HeaderPrefix  string `json:"header_prefix"`
	RewardAddress string `json:"reward_address"`
}

type SubmitResult struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func ValidateRewardAddress(address string) error {
	got := strings.TrimSpace(address)
	if !rewardAddressPattern.MatchString(got) {
		return fmt.Errorf("wallet address must be 40 lowercase hex characters")
	}
	return nil
}

func Resolve(cfg Config) (Config, error) {
	if err := ValidateRewardAddress(cfg.Address); err != nil {
		return Config{}, err
	}
	if err := params.ValidateMiningBackend(cfg.Backend); err != nil {
		return Config{}, err
	}
	network := params.NormalizeMiningNetwork(cfg.Network)
	rpcURL := strings.TrimSpace(cfg.RPCURL)
	if rpcURL == "" {
		resolved, err := params.MiningRPCForNetwork(network)
		if err != nil {
			return Config{}, err
		}
		rpcURL = resolved
	} else if network == params.NetworkMainnet && !params.MainnetMiningAuthorized {
		return Config{}, fmt.Errorf("mainnet mining is not authorized; Sudharma mainnet remains GPU-only and closed until launch")
	}
	backend := strings.TrimSpace(cfg.Backend)
	if backend == "" {
		backend = params.ProductionMiningBackend
	}
	return Config{
		Address: strings.TrimSpace(cfg.Address),
		Network: network,
		RPCURL:  strings.TrimRight(rpcURL, "/"),
		Backend: backend,
		Device:  cfg.Device,
	}, nil
}

func WorkFromJSON(raw []byte) (Work, error) {
	var work Work
	if err := json.Unmarshal(raw, &work); err != nil {
		return Work{}, fmt.Errorf("invalid mining work JSON: %w", err)
	}
	if err := params.ValidateProductionMiningAlgorithm(work.Algorithm); err != nil {
		return Work{}, err
	}
	if work.Version != 0 && work.Version != 2 {
		return Work{}, fmt.Errorf("%s (unsupported work version %d)", params.GPUOnlyMiningMessage, work.Version)
	}
	if strings.TrimSpace(work.HeaderPrefix) == "" || strings.TrimSpace(work.Target) == "" {
		return Work{}, fmt.Errorf("mining work is missing GPU-PoW header or target")
	}
	return work, nil
}
