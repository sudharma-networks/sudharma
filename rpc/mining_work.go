package rpc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/pow"
)

var miningWorkIDDomain = []byte("SUDHARMA-GPU-POW-V1-WORK-ID\x00")

// MiningWorkTemplate is the constrained, read-only contract an external miner
// needs to search a Version-2 Sudharma block. Nonce is intentionally excluded:
// it is the only field miners are allowed to vary when submitting a solution.
type MiningWorkTemplate struct {
	WorkID          string `json:"work_id"`
	Algorithm       string `json:"algorithm"`
	Version         uint32 `json:"version"`
	Height          uint64 `json:"height"`
	Difficulty      uint32 `json:"difficulty"`
	TargetHex       string `json:"target"`
	HeaderPrefixHex string `json:"header_prefix"`
	RewardAddress   string `json:"reward_address"`
}

// NewMiningWorkTemplate snapshots the immutable consensus fields for external
// GPU mining. It does not activate GPU-PoW, choose a cache size, or expose any
// administrative node operation.
func NewMiningWorkTemplate(block *blockchain.Block) (MiningWorkTemplate, error) {
	if block == nil {
		return MiningWorkTemplate{}, fmt.Errorf("mining block cannot be nil")
	}
	if block.Version != 2 {
		return MiningWorkTemplate{}, fmt.Errorf("external GPU-PoW work requires block version 2")
	}
	if block.Difficulty == 0 {
		return MiningWorkTemplate{}, fmt.Errorf("mining difficulty must be positive")
	}
	if strings.TrimSpace(block.MinerAddress) == "" {
		return MiningWorkTemplate{}, fmt.Errorf("mining reward address is required")
	}

	headerWithNonce := block.HeaderBytes(0)
	if len(headerWithNonce) < 8 {
		return MiningWorkTemplate{}, fmt.Errorf("invalid canonical block header")
	}
	headerPrefix := append([]byte(nil), headerWithNonce[:len(headerWithNonce)-8]...)

	target := pow.TargetFromDifficulty(block.Difficulty)
	if target == nil || target.Sign() <= 0 || target.BitLen() > 256 {
		return MiningWorkTemplate{}, fmt.Errorf("invalid proof-of-work target")
	}
	targetBytes := make([]byte, 32)
	target.FillBytes(targetBytes)

	workInput := make([]byte, 0, len(miningWorkIDDomain)+len(pow.GPUV1AlgorithmID)+len(headerPrefix))
	workInput = append(workInput, miningWorkIDDomain...)
	workInput = append(workInput, pow.GPUV1AlgorithmID...)
	workInput = append(workInput, headerPrefix...)
	workID := sha256.Sum256(workInput)

	return MiningWorkTemplate{
		WorkID:          hex.EncodeToString(workID[:]),
		Algorithm:       pow.GPUV1AlgorithmID,
		Version:         block.Version,
		Height:          block.Height,
		Difficulty:      block.Difficulty,
		TargetHex:       hex.EncodeToString(targetBytes),
		HeaderPrefixHex: hex.EncodeToString(headerPrefix),
		RewardAddress:   block.MinerAddress,
	}, nil
}
