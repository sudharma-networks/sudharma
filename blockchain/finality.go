package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
)

// Confirmations returns the number of blocks from blockHeight through the
// current chain tip, counting the block itself as one confirmation.
func Confirmations(chain *Chain, blockHeight uint64) (uint64, error) {
	if chain == nil {
		return 0, fmt.Errorf("chain cannot be nil")
	}

	tipHeight := chain.Height()
	if blockHeight > tipHeight {
		return 0, fmt.Errorf("block height %d is above chain tip %d", blockHeight, tipHeight)
	}

	return tipHeight - blockHeight + 1, nil
}

// FinalizedHeight returns the highest block height that is outside the
// automatic-reorganization window.
//
// Sudharma uses Proof of Work, so this is local protocol finality rather than
// mathematical finality: a block is called finalized when replacing it would
// require a reorganization deeper than MaxAutomaticReorgDepth, which normal
// nodes refuse automatically.
//
// Genesis is always treated as finalized.
func FinalizedHeight(chain *Chain) (uint64, error) {
	if chain == nil {
		return 0, fmt.Errorf("chain cannot be nil")
	}

	tipHeight := chain.Height()
	if tipHeight <= params.MaxAutomaticReorgDepth {
		return 0, nil
	}

	return tipHeight - params.MaxAutomaticReorgDepth, nil
}

// IsBlockFinalized reports whether blockHeight is at or below the current
// deterministic finalized height.
func IsBlockFinalized(chain *Chain, blockHeight uint64) (bool, error) {
	if chain == nil {
		return false, fmt.Errorf("chain cannot be nil")
	}

	if blockHeight > chain.Height() {
		return false, fmt.Errorf("block height %d is above chain tip %d", blockHeight, chain.Height())
	}

	finalizedHeight, err := FinalizedHeight(chain)
	if err != nil {
		return false, err
	}

	return blockHeight <= finalizedHeight, nil
}
