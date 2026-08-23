package blockchain

import "fmt"

// ValidateFinalizedReorg ensures a candidate chain does not replace any block
// at or below the local finalized height. This makes the finality boundary an
// explicit reorganization invariant instead of relying only on a depth check.
//
// Sudharma finality is operational Proof-of-Work finality: it is derived from
// the automatic reorganization window and does not claim mathematical or BFT
// irreversibility.
func ValidateFinalizedReorg(current *Chain, candidate *Chain) error {
	if current == nil {
		return fmt.Errorf("current chain cannot be nil")
	}
	if candidate == nil {
		return fmt.Errorf("candidate chain cannot be nil")
	}

	finalizedHeight, err := FinalizedHeight(current)
	if err != nil {
		return fmt.Errorf("failed calculating finalized height: %w", err)
	}

	_, commonHeight, err := ReorgDepth(current, candidate)
	if err != nil {
		return fmt.Errorf("failed calculating reorg common height: %w", err)
	}

	if commonHeight < finalizedHeight {
		return fmt.Errorf(
			"candidate would replace finalized history: common height %d is below finalized height %d",
			commonHeight,
			finalizedHeight,
		)
	}

	return nil
}
