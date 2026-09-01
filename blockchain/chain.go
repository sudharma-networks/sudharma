package blockchain

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

type Chain struct {
	mu        sync.RWMutex
	blocks    []*Block
	totalWork *big.Int
}

func NewChain() *Chain {
	chain, err := NewChainFor(params.NetworkPublicTestnet)
	if err != nil {
		panic(err)
	}
	return chain
}

// NewChainFor creates a chain whose genesis matches the requested network.
func NewChainFor(network params.NetworkID) (*Chain, error) {
	genesis, err := GenesisFor(network)
	if err != nil {
		return nil, err
	}
	return &Chain{
		blocks: []*Block{
			genesis,
		},
		totalWork: blockWork(genesis.Difficulty),
	}, nil
}

// ValidateChainGenesis ensures an on-disk chain belongs to the active network.
func ValidateChainGenesis(chain *Chain, network params.NetworkID) error {
	if chain == nil {
		return fmt.Errorf("chain cannot be nil")
	}
	genesis, ok := chain.BlockByHeight(0)
	if !ok || genesis == nil {
		return fmt.Errorf("chain missing genesis block")
	}
	expected, err := GenesisFor(network)
	if err != nil {
		return err
	}
	if genesis.Hash() != expected.Hash() {
		return fmt.Errorf("chain genesis does not match network %q", network)
	}
	return nil
}

func (c *Chain) Height() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.blocks) == 0 {
		return 0
	}

	return c.blocks[len(c.blocks)-1].Height
}

func (c *Chain) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.blocks)
}

func (c *Chain) Tip() *Block {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.blocks) == 0 {
		return nil
	}

	return c.blocks[len(c.blocks)-1]
}

func (c *Chain) BlockByHeight(height uint64) (*Block, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if height >= uint64(len(c.blocks)) {
		return nil, false
	}

	block := c.blocks[height]
	if block.Height != height {
		return nil, false
	}

	return block, true
}

// TotalWork returns an independent copy of the cumulative Proof-of-Work value.
func (c *Chain) TotalWork() *big.Int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return new(big.Int).Set(c.totalWork)
}

// AddBlock validates and appends a block, then adds its work to cumulative chain work.
func (c *Chain) AddBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.blocks) == 0 {
		return fmt.Errorf("chain has no genesis block")
	}

	previous := c.blocks[len(c.blocks)-1]
	expectedDifficulty := consensus.NextDifficultyFromHistory(
		previous.Difficulty,
		recentBlockIntervalsLocked(c.blocks),
	)

	if block.Difficulty != expectedDifficulty {
		return fmt.Errorf(
			"block validation failed: invalid history-based block difficulty: expected %d, got %d",
			expectedDifficulty,
			block.Difficulty,
		)
	}

	if err := validateBlockCore(block, previous); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	c.blocks = append(c.blocks, block)
	c.totalWork.Add(c.totalWork, blockWork(block.Difficulty))

	return nil
}

// blockWork returns exact target-derived Proof-of-Work represented by a block.
func blockWork(difficulty uint32) *big.Int {
	return consensus.WorkFromDifficulty(difficulty)
}
