package blockchain

import "fmt"

// ValidateAndCloneChain rebuilds a chain from canonical genesis and validates
// every non-genesis block through the normal history-aware admission path.
// The returned chain has cumulative work recomputed locally from validated
// blocks, so callers do not need to trust cached work stored on the source.
func ValidateAndCloneChain(source *Chain) (*Chain, error) {
	if source == nil {
		return nil, fmt.Errorf("source chain cannot be nil")
	}

	source.mu.RLock()
	if len(source.blocks) == 0 {
		source.mu.RUnlock()
		return nil, fmt.Errorf("source chain has no blocks")
	}
	blocks := make([]*Block, len(source.blocks))
	copy(blocks, source.blocks)
	source.mu.RUnlock()

	expectedGenesis := NewGenesisBlock()
	if blocks[0] == nil {
		return nil, fmt.Errorf("source genesis block is nil")
	}
	if blocks[0].Hash() != expectedGenesis.Hash() {
		return nil, fmt.Errorf("source has wrong genesis block")
	}

	validated := NewChain()
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]
		if block == nil {
			return nil, fmt.Errorf("source block %d is nil", i)
		}
		if err := validated.AddBlock(block); err != nil {
			return nil, fmt.Errorf("source block %d failed validation: %w", block.Height, err)
		}
	}

	return validated, nil
}
