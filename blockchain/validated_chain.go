package blockchain

import "fmt"

// ValidateAndCloneChain rebuilds a chain from its network's canonical genesis
// and validates every non-genesis block through the normal history-aware
// admission path. The returned chain preserves the source network identity and
// recomputes cumulative work locally from validated blocks.
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
	network := source.network
	source.mu.RUnlock()

	if blocks[0] == nil {
		return nil, fmt.Errorf("source genesis block is nil")
	}
	validated, err := newChainFromGenesisForNetwork(network, blocks[0])
	if err != nil {
		return nil, fmt.Errorf("source has wrong genesis block: %w", err)
	}

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
