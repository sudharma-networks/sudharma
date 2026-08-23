package blockchain

import (
	"fmt"
	"math/big"
	"sync"
)

type Chain struct {
	mu        sync.RWMutex
	blocks    []*Block
	totalWork *big.Int
}

func NewChain() *Chain {
	genesis := NewGenesisBlock()

	return &Chain{
		blocks: []*Block{
			genesis,
		},
		totalWork: blockWork(genesis.Difficulty),
	}
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

func (c *Chain) BlockByHeight(
	height uint64,
) (*Block, bool) {

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

// TotalWork returns an independent copy of the
// cumulative Proof-of-Work value.
func (c *Chain) TotalWork() *big.Int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return new(big.Int).Set(c.totalWork)
}

// AddBlock validates and appends a block,
// then adds its work to cumulative chain work.
func (c *Chain) AddBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.blocks) == 0 {
		return fmt.Errorf(
			"chain has no genesis block",
		)
	}

	previous :=
		c.blocks[len(c.blocks)-1]

	if err := ValidateBlockBasic(
		block,
		previous,
	); err != nil {
		return fmt.Errorf(
			"block validation failed: %w",
			err,
		)
	}

	c.blocks = append(
		c.blocks,
		block,
	)

	work := blockWork(
		block.Difficulty,
	)

	c.totalWork.Add(
		c.totalWork,
		work,
	)

	return nil
}

// blockWork estimates the amount of Proof-of-Work
// represented by a block at the given difficulty.
//
// For our current Sudharma Network development difficulty model,
// work is proportional to difficulty.
//
// Later, when the final PoW target format is locked,
// we can use exact target-derived chain work.
func blockWork(
	difficulty uint32,
) *big.Int {

	if difficulty == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).SetUint64(
		uint64(difficulty),
	)
}
