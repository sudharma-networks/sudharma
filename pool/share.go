package pool

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/pow"
)

// ShareKind classifies a submitted nonce against pool and network difficulty.
type ShareKind int

const (
	ShareInvalid ShareKind = iota
	ShareValid
	ShareBlock
)

func (k ShareKind) String() string {
	switch k {
	case ShareValid:
		return "valid"
	case ShareBlock:
		return "block"
	default:
		return "invalid"
	}
}

// ShareResult is the outcome of validating a worker nonce.
type ShareResult struct {
	Kind            ShareKind
	Hash            string
	Nonce           uint64
	PoolDifficulty  uint32
	BlockDifficulty uint32
	ShareWork       uint64
	BlockWork       uint64
}

// ValidateShare checks a nonce against pool difficulty and network block difficulty.
// Pool difficulty must be <= block difficulty.
func ValidateShare(block *blockchain.Block, nonce uint64, poolDifficulty, blockDifficulty uint32) (ShareResult, error) {
	if block == nil {
		return ShareResult{}, fmt.Errorf("missing block template")
	}
	if poolDifficulty == 0 || blockDifficulty == 0 {
		return ShareResult{}, fmt.Errorf("difficulty must be >= 1")
	}
	if poolDifficulty > blockDifficulty {
		return ShareResult{}, fmt.Errorf("pool difficulty %d exceeds block difficulty %d", poolDifficulty, blockDifficulty)
	}

	hash := pow.HashBlock(block, nonce)
	result := ShareResult{
		Hash:            hash,
		Nonce:           nonce,
		PoolDifficulty:  poolDifficulty,
		BlockDifficulty: blockDifficulty,
		ShareWork:       workUnits(poolDifficulty),
		BlockWork:       workUnits(blockDifficulty),
	}

	if pow.ValidHash(hash, blockDifficulty) {
		result.Kind = ShareBlock
		return result, nil
	}
	if pow.ValidHash(hash, poolDifficulty) {
		result.Kind = ShareValid
		return result, nil
	}
	return result, nil
}

// ShareValue estimates the expected block-reward fraction for one valid pool share.
func ShareValue(blockReward uint64, poolDifficulty, blockDifficulty uint32) uint64 {
	if blockReward == 0 || poolDifficulty == 0 || blockDifficulty == 0 {
		return 0
	}
	return (blockReward * uint64(poolDifficulty)) / uint64(blockDifficulty)
}

// TargetHex returns the hex target for a difficulty level.
func TargetHex(difficulty uint32) string {
	return consensus.TargetFromDifficulty(difficulty).Text(16)
}

func workUnits(difficulty uint32) uint64 {
	work := consensus.WorkFromDifficulty(difficulty)
	if !work.IsUint64() {
		return ^uint64(0)
	}
	return work.Uint64()
}
