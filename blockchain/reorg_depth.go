package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
)

// ReorgDepth returns how many blocks from the current active chain
// would be replaced if candidate were adopted.

// It also returns the height of the last common block.
func ReorgDepth(current *Chain, candidate *Chain) (depth uint64, commonHeight uint64, err error) {
	if current == nil {
		return 0, 0, fmt.Errorf("current chain cannot be nil")
	}

	if candidate == nil {
		return 0, 0, fmt.Errorf("candidate chain cannot be nil")
	}

	currentHeight := current.Height()
	candidateHeight := candidate.Height()

	maxCommon := currentHeight
	if candidateHeight < maxCommon {
		maxCommon = candidateHeight
	}

	foundCommon := false

	for height := maxCommon; ; height-- {
		currentBlock, currentOK := current.BlockByHeight(height)
		candidateBlock, candidateOK := candidate.BlockByHeight(height)

		if currentOK && candidateOK &&
			currentBlock != nil && candidateBlock != nil &&
			currentBlock.Hash() == candidateBlock.Hash() {

			commonHeight = height
			foundCommon = true
			break
		}

		if height == 0 {
			break
		}
	}

	if !foundCommon {
		return 0, 0, fmt.Errorf("chains do not share a common genesis")
	}

	if currentHeight <= commonHeight {
		return 0, commonHeight, nil
	}

	return currentHeight - commonHeight, commonHeight, nil
}

// ValidateAutomaticReorgDepth enforces the consensus limit on
// how much confirmed history may be replaced automatically.
func ValidateAutomaticReorgDepth(depth uint64) error {
	if depth > params.MaxAutomaticReorgDepth {
		return fmt.Errorf(
			"reorganization depth %d exceeds maximum automatic depth %d",
			depth,
			params.MaxAutomaticReorgDepth,
		)
	}

	return nil
}
