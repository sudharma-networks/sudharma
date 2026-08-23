package blockchain

import "fmt"

// FinalityStatus is a stable read model for node/RPC callers. It exposes
// finality information without requiring callers to duplicate consensus math.
type FinalityStatus struct {
	TipHeight       uint64
	FinalizedHeight uint64
	ReorgWindow     uint64
}

// CurrentFinalityStatus returns the active-chain tip, finalized height and
// the number of blocks currently inside the automatic reorganization window.
func CurrentFinalityStatus(chain *Chain) (FinalityStatus, error) {
	if chain == nil {
		return FinalityStatus{}, fmt.Errorf("chain cannot be nil")
	}

	tipHeight := chain.Height()
	finalizedHeight, err := FinalizedHeight(chain)
	if err != nil {
		return FinalityStatus{}, err
	}

	return FinalityStatus{
		TipHeight:       tipHeight,
		FinalizedHeight: finalizedHeight,
		ReorgWindow:     tipHeight - finalizedHeight,
	}, nil
}

// BlockFinalityStatus describes confirmation/finality information for one
// active-chain block height.
type BlockFinalityStatus struct {
	Height        uint64
	Confirmations uint64
	Finalized     bool
}

// FinalityStatusForBlock returns confirmation count and finality for one
// active-chain block height using the same consensus helpers used elsewhere.
func FinalityStatusForBlock(chain *Chain, blockHeight uint64) (BlockFinalityStatus, error) {
	confirmations, err := Confirmations(chain, blockHeight)
	if err != nil {
		return BlockFinalityStatus{}, err
	}

	finalized, err := IsBlockFinalized(chain, blockHeight)
	if err != nil {
		return BlockFinalityStatus{}, err
	}

	return BlockFinalityStatus{
		Height:        blockHeight,
		Confirmations: confirmations,
		Finalized:     finalized,
	}, nil
}
