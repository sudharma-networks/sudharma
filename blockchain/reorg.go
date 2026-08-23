package blockchain

import (
	"fmt"
	"math/big"
)

// CloneChain creates an independent Chain structure containing
// the same block sequence and cumulative work.
//
// Blocks are treated as immutable consensus objects once accepted.
func CloneChain(source *Chain) (*Chain, error) {
	if source == nil {
		return nil, fmt.Errorf("source chain cannot be nil")
	}

	source.mu.RLock()
	defer source.mu.RUnlock()

	if len(source.blocks) == 0 {
		return nil, fmt.Errorf("source chain has no blocks")
	}

	blocks := make([]*Block, len(source.blocks))
	copy(blocks, source.blocks)

	return &Chain{
		blocks:    blocks,
		totalWork: new(big.Int).Set(source.totalWork),
	}, nil
}

// ReplaceWith atomically replaces the current chain with another fully
// revalidated candidate chain. BetterChain must be used before replacement.
func (c *Chain) ReplaceWith(candidate *Chain) error {
	if c == nil {
		return fmt.Errorf("current chain cannot be nil")
	}
	if candidate == nil {
		return fmt.Errorf("candidate chain cannot be nil")
	}
	if c == candidate {
		return nil
	}

	candidate.mu.RLock()
	if len(candidate.blocks) == 0 {
		candidate.mu.RUnlock()
		return fmt.Errorf("candidate chain has no blocks")
	}

	newBlocks := make([]*Block, len(candidate.blocks))
	copy(newBlocks, candidate.blocks)
	candidate.mu.RUnlock()

	// Candidate must contain the canonical Sudharma Network genesis.
	expectedGenesis := NewGenesisBlock()
	if newBlocks[0] == nil {
		return fmt.Errorf("candidate genesis block is nil")
	}
	if newBlocks[0].Hash() != expectedGenesis.Hash() {
		return fmt.Errorf("candidate has wrong genesis block")
	}

	// Rebuild a fresh validation chain from canonical genesis. This ensures
	// every candidate block obeys the same history-derived difficulty rules
	// used by normal block admission and recomputes cumulative work locally.
	validated := NewChain()
	for i := 1; i < len(newBlocks); i++ {
		block := newBlocks[i]
		if block == nil {
			return fmt.Errorf("candidate block %d is nil", i)
		}
		if err := validated.AddBlock(block); err != nil {
			return fmt.Errorf("candidate block %d failed validation: %w", block.Height, err)
		}
	}

	newTotalWork := validated.TotalWork()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.blocks = newBlocks
	c.totalWork = newTotalWork
	return nil
}

// BuildStateFromChain deterministically recreates the complete
// confirmed account state by replaying every non-genesis block.
//
// This is used during chain reorganization so state always matches
// the selected blockchain history.
func BuildStateFromChain(chain *Chain) (*State, error) {
	if chain == nil {
		return nil, fmt.Errorf("chain cannot be nil")
	}

	state := NewState()
	height := chain.Height()

	for blockHeight := uint64(1); blockHeight <= height; blockHeight++ {
		block, ok := chain.BlockByHeight(blockHeight)
		if !ok {
			return nil, fmt.Errorf("block %d is missing", blockHeight)
		}
		if block == nil {
			return nil, fmt.Errorf("block %d is nil", blockHeight)
		}
		if block.MinerAddress == "" {
			return nil, fmt.Errorf("block %d has no miner address", blockHeight)
		}

		if _, err := ProcessBlock(state, block, block.MinerAddress); err != nil {
			return nil, fmt.Errorf("failed replaying block %d: %w", blockHeight, err)
		}
	}

	return state, nil
}

// ReorganizeToCandidate selects and atomically installs a better chain
// together with the state produced by that chain.
//
// adopted is true only when candidate replaces current.
func ReorganizeToCandidate(current *Chain, currentState *State, candidate *Chain) (adopted bool, err error) {
	if current == nil {
		return false, fmt.Errorf("current chain cannot be nil")
	}
	if currentState == nil {
		return false, fmt.Errorf("current state cannot be nil")
	}
	if candidate == nil {
		return false, fmt.Errorf("candidate chain cannot be nil")
	}

	best, err := BetterChain(current, candidate)
	if err != nil {
		return false, err
	}
	if best == current {
		return false, nil
	}

	reorgDepth, commonHeight, err := ReorgDepth(current, candidate)
	if err != nil {
		return false, fmt.Errorf("failed calculating reorg depth: %w", err)
	}
	if err := ValidateFinalizedReorg(current, candidate); err != nil {
		return false, err
	}
	if err := ValidateAutomaticReorgDepth(reorgDepth); err != nil {
		return false, fmt.Errorf("%w (common height %d)", err, commonHeight)
	}

	// Build candidate state completely before changing live data.
	candidateState, err := BuildStateFromChain(candidate)
	if err != nil {
		return false, fmt.Errorf("candidate state rebuild failed: %w", err)
	}

	// Keep backups in case state replacement fails.
	originalChain, err := CloneChain(current)
	if err != nil {
		return false, err
	}
	originalState := currentState.Clone()

	if err := current.ReplaceWith(candidate); err != nil {
		return false, fmt.Errorf("failed replacing chain: %w", err)
	}

	if err := currentState.ReplaceWith(candidateState); err != nil {
		// Best-effort rollback.
		_ = current.ReplaceWith(originalChain)
		_ = currentState.ReplaceWith(originalState)
		return false, fmt.Errorf("failed replacing blockchain state: %w", err)
	}

	return true, nil
}
