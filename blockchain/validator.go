package blockchain

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/sudharma-networks/sudharma/consensus"
)

const (
	// A block timestamp may be at most
	// two hours ahead of the node's local time.
	MaxFutureBlockSeconds int64 = 2 * 60 * 60
)

// ValidateBlockBasic performs consensus-critical
// structural and Proof-of-Work validation.
func ValidateBlockBasic(
	block *Block,
	previous *Block,
) error {

	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	if previous == nil {
		return fmt.Errorf("previous block cannot be nil")
	}

	// ------------------------------------------------
	// Block height
	// ------------------------------------------------

	expectedHeight := previous.Height + 1

	if block.Height != expectedHeight {
		return fmt.Errorf(
			"invalid block height: expected %d, got %d",
			expectedHeight,
			block.Height,
		)
	}

	// ------------------------------------------------
	// Previous block hash
	// ------------------------------------------------

	expectedPreviousHash := previous.Hash()

	if block.PreviousHash != expectedPreviousHash {
		return fmt.Errorf(
			"invalid previous hash",
		)
	}

	// ------------------------------------------------
	// Timestamp
	// ------------------------------------------------

	if block.Timestamp <= previous.Timestamp {
		return fmt.Errorf(
			"invalid block timestamp: must be greater than previous block",
		)
	}

	now := time.Now().Unix()

	if block.Timestamp > now+MaxFutureBlockSeconds {
		return fmt.Errorf(
			"invalid block timestamp: too far in the future",
		)
	}

	// ------------------------------------------------
	// Difficulty
	// ------------------------------------------------

	actualBlockTime :=
		block.Timestamp - previous.Timestamp

	expectedDifficulty :=
		consensus.NextDifficulty(
			previous.Difficulty,
			actualBlockTime,
		)

	if block.Difficulty != expectedDifficulty {
		return fmt.Errorf(
			"invalid block difficulty: expected %d, got %d",
			expectedDifficulty,
			block.Difficulty,
		)
	}

	// ------------------------------------------------
	// Merkle root
	// ------------------------------------------------

	expectedMerkleRoot :=
		block.CalculateMerkleRoot()

	if block.MerkleRoot != expectedMerkleRoot {
		return fmt.Errorf(
			"invalid merkle root",
		)
	}

	// ------------------------------------------------
	// Proof of Work
	// ------------------------------------------------

	if !validBlockProofOfWork(block) {
		return fmt.Errorf(
			"invalid proof of work",
		)
	}

	return nil
}

// validBlockProofOfWork checks whether the block hash
// satisfies the declared difficulty.
func validBlockProofOfWork(block *Block) bool {
	hashBytes, err :=
		hex.DecodeString(block.Hash())

	if err != nil {
		return false
	}

	hashInt :=
		new(big.Int).SetBytes(hashBytes)

	target :=
		blockTargetFromDifficulty(
			block.Difficulty,
		)

	return hashInt.Cmp(target) <= 0
}

// blockTargetFromDifficulty converts Sudharma Network's
// development difficulty into a 256-bit PoW target.
//
// Higher difficulty:
// smaller target
// harder mining.
func blockTargetFromDifficulty(
	difficulty uint32,
) *big.Int {

	return consensus.TargetFromDifficulty(
		difficulty,
	)
}
