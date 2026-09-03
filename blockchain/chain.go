package blockchain

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

type Chain struct {
	mu            sync.RWMutex
	network       params.NetworkID
	blocks        []*Block
	totalWork     *big.Int
	powPolicy     PoWPolicy
	proofVerifier ProofVerifier
}

func NewChain() *Chain {
	chain, err := NewChainFor(params.NetworkPublicTestnet)
	if err != nil {
		panic(err)
	}
	return chain
}

// NewChainFor creates a chain whose immutable identity, genesis, and proof
// policy match the requested network. Both network policies remain legacy-only
// until a later dedicated activation change.
func NewChainFor(network params.NetworkID) (*Chain, error) {
	policy, err := PoWPolicyForNetwork(network)
	if err != nil {
		return nil, err
	}
	return NewChainForWithConsensus(network, policy, legacyProofVerifier{})
}

// NewChainForWithConsensus creates a runtime chain using an explicit immutable
// proof policy and verifier. It still observes the existing GenesisFor launch
// gate, so it cannot be used to bypass unauthorized mainnet startup.
func NewChainForWithConsensus(
	network params.NetworkID,
	policy PoWPolicy,
	verifier ProofVerifier,
) (*Chain, error) {
	genesis, err := GenesisFor(network)
	if err != nil {
		return nil, err
	}
	return newChainFromGenesisForNetworkWithConsensus(network, genesis, policy, verifier)
}

// newChainFromGenesisForNetwork builds a validation-only chain from the
// canonical genesis for network. Unlike NewChainFor, it does not authorize
// runtime mainnet launch; callers must already possess the expected genesis.
func newChainFromGenesisForNetwork(network params.NetworkID, genesis *Block) (*Chain, error) {
	policy, err := PoWPolicyForNetwork(network)
	if err != nil {
		return nil, err
	}
	return newChainFromGenesisForNetworkWithConsensus(
		network,
		genesis,
		policy,
		legacyProofVerifier{},
	)
}

// newChainFromGenesisForNetworkWithConsensus reconstructs a validation-only
// chain under explicit consensus configuration. The verifier must support every
// block version the policy can select, otherwise construction fails closed.
func newChainFromGenesisForNetworkWithConsensus(
	network params.NetworkID,
	genesis *Block,
	policy PoWPolicy,
	verifier ProofVerifier,
) (*Chain, error) {
	if genesis == nil {
		return nil, fmt.Errorf("genesis block cannot be nil")
	}
	if _, err := params.MonetaryPolicyFor(network); err != nil {
		return nil, err
	}
	if verifier == nil {
		return nil, fmt.Errorf("proof verifier cannot be nil")
	}
	if !verifier.SupportsVersion(1) {
		return nil, fmt.Errorf("proof verifier does not support legacy block Version 1")
	}
	if policy.GPUV1ActivationHeight != params.GPUV1ActivationDisabled &&
		!verifier.SupportsVersion(2) {
		return nil, fmt.Errorf("proof verifier does not support configured GPU-PoW Version 2 activation")
	}
	if !policy.VersionAllowed(genesis.Version, genesis.Height) {
		expectedVersion, _ := policy.VersionAtHeight(genesis.Height)
		return nil, fmt.Errorf(
			"genesis block version %d does not match proof-of-work policy version %d",
			genesis.Version,
			expectedVersion,
		)
	}

	var expected *Block
	switch network {
	case params.NetworkPublicTestnet:
		expected = NewGenesisBlock()
	case params.NetworkMainnet:
		expected = NewMainnetGenesisBlock()
	default:
		return nil, fmt.Errorf("unknown network %q", network)
	}
	if genesis.Hash() != expected.Hash() {
		return nil, fmt.Errorf("genesis block does not match network %q", network)
	}

	return &Chain{
		network: network,
		blocks: []*Block{
			genesis,
		},
		totalWork:     blockWork(genesis.Difficulty),
		powPolicy:     policy,
		proofVerifier: verifier,
	}, nil
}

// Network returns the immutable network identity bound to this chain.
func (c *Chain) Network() params.NetworkID {
	if c == nil {
		return ""
	}
	return c.network
}

// PoWPolicy returns the immutable proof-version policy bound to this chain.
func (c *Chain) PoWPolicy() PoWPolicy {
	if c == nil {
		return PoWPolicy{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.powPolicy
}

// proofValidationConfig returns the immutable proof configuration. The
// interface reference is copied; verifier implementations are responsible for
// synchronizing any internal caches they may add in later stages.
func (c *Chain) proofValidationConfig() (PoWPolicy, ProofVerifier) {
	if c == nil {
		return PoWPolicy{}, nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.powPolicy, c.proofVerifier
}

// MonetaryPolicy returns the monetary policy implied by the chain's network identity.
func (c *Chain) MonetaryPolicy() (params.MonetaryPolicy, error) {
	if c == nil {
		return 0, fmt.Errorf("chain cannot be nil")
	}
	return params.MonetaryPolicyFor(c.network)
}

// ValidateChainGenesis ensures an on-disk chain belongs to the active network.
func ValidateChainGenesis(chain *Chain, network params.NetworkID) error {
	if chain == nil {
		return fmt.Errorf("chain cannot be nil")
	}
	if chain.Network() != network {
		return fmt.Errorf("chain identity does not match network %q", network)
	}
	genesis, ok := chain.BlockByHeight(0)
	if !ok || genesis == nil {
		return fmt.Errorf("chain missing genesis block")
	}

	var expected *Block
	switch network {
	case params.NetworkPublicTestnet:
		expected = NewGenesisBlock()
	case params.NetworkMainnet:
		expected = NewMainnetGenesisBlock()
	default:
		return fmt.Errorf("unknown network %q", network)
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
