package blockchain

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/sudharma-networks/sudharma/consensus"
)

type Chain struct {
	mu            sync.RWMutex
	blocks        []*Block
	totalWork     *big.Int
	powPolicy     PoWPolicy
	proofVerifier ProofVerifier
}

func NewChain() *Chain {
	chain, err := NewChainWithConsensus(LegacyOnlyPoWPolicy(), legacyProofVerifier{})
	if err != nil {
		panic(err)
	}
	return chain
}

// NewChainWithConsensus constructs a chain with an immutable version policy
// and a verifier capable of every version that policy may select.
func NewChainWithConsensus(policy PoWPolicy, verifier ProofVerifier) (*Chain, error) {
	if verifier == nil {
		return nil, fmt.Errorf("proof verifier cannot be nil")
	}
	if !verifier.SupportsVersion(1) {
		return nil, fmt.Errorf("proof verifier does not support legacy Version 1")
	}
	if policy.GPUV1ActivationHeight != LegacyOnlyPoWPolicy().GPUV1ActivationHeight &&
		!verifier.SupportsVersion(2) {
		return nil, fmt.Errorf("finite GPU-PoW activation requires Version 2 verification support")
	}

	genesis := NewGenesisBlock()

	return &Chain{
		blocks: []*Block{
			genesis,
		},
		totalWork:     blockWork(genesis.Difficulty),
		powPolicy:     policy,
		proofVerifier: verifier,
	}, nil
}

// PoWPolicy returns a copy of the chain's immutable proof-of-work policy.
func (c *Chain) PoWPolicy() PoWPolicy {
	if c == nil {
		return LegacyOnlyPoWPolicy()
	}
	return c.powPolicy
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

	if err := validateBlockCoreWithProof(
		block,
		previous,
		c.powPolicy,
		c.proofVerifier,
	); err != nil {
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
