package blockchain

import "fmt"

// ValidateBlockAgainstChain performs history-aware consensus validation.
func ValidateBlockAgainstChain(chain *Chain, block *Block) error {
	if chain == nil {
		return fmt.Errorf("chain cannot be nil")
	}
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	previous := chain.Tip()
	if previous == nil {
		return fmt.Errorf("chain tip cannot be nil")
	}

	expectedDifficulty, err := ExpectedNextDifficulty(chain)
	if err != nil {
		return fmt.Errorf("failed calculating expected difficulty: %w", err)
	}
	if block.Difficulty != expectedDifficulty {
		return fmt.Errorf(
			"invalid history-based block difficulty: expected %d, got %d",
			expectedDifficulty,
			block.Difficulty,
		)
	}

	return validateBlockCoreWithProof(
		block,
		previous,
		chain.powPolicy,
		chain.proofVerifier,
	)
}
