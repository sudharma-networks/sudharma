package blockchain

import (
	"fmt"
	"math/big"

	"github.com/sudharma-networks/sudharma/consensus"
)

const MaxFutureBlockSeconds int64 = 2 * 60 * 60

// ValidateBlockBasic preserves previous-block validation for callers and tests
// that do not have the full chain available. Consensus admission paths should
// prefer ValidateBlockAgainstChain so difficulty is derived from chain history.
func ValidateBlockBasic(block *Block, previous *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}
	if previous == nil {
		return fmt.Errorf("previous block cannot be nil")
	}

	actualBlockTime := block.Timestamp - previous.Timestamp
	expectedDifficulty := consensus.NextDifficulty(previous.Difficulty, actualBlockTime)
	if block.Difficulty != expectedDifficulty {
		return fmt.Errorf(
			"invalid block difficulty: expected %d, got %d",
			expectedDifficulty,
			block.Difficulty,
		)
	}

	return validateBlockCore(block, previous)
}

func validBlockProofOfWork(block *Block) bool {
	return validBlockProofOfWorkCore(block)
}

func blockTargetFromDifficulty(difficulty uint32) *big.Int {
	return consensus.TargetFromDifficulty(difficulty)
}
