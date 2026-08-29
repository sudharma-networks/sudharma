package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveToFile saves the complete Sudharma Network chain to disk.
func (c *Chain) SaveToFile(path string) error {
	if c == nil {
		return fmt.Errorf("chain cannot be nil")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if path == "" {
		return fmt.Errorf("storage path cannot be empty")
	}

	dir := filepath.Dir(path)

	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf(
				"failed to create storage directory: %w",
				err,
			)
		}
	}

	data, err := json.MarshalIndent(
		c.blocks,
		"",
		"  ",
	)

	if err != nil {
		return fmt.Errorf(
			"failed to encode blockchain: %w",
			err,
		)
	}

	tempPath := path + ".tmp"

	if err := os.WriteFile(
		tempPath,
		data,
		0644,
	); err != nil {
		return fmt.Errorf(
			"failed to write blockchain: %w",
			err,
		)
	}

	if err := os.Rename(
		tempPath,
		path,
	); err != nil {
		return fmt.Errorf(
			"failed to finalize blockchain file: %w",
			err,
		)
	}

	return nil
}

// LoadChainFromFile loads and validates a Sudharma Network chain from disk.
func LoadChainFromFile(path string) (*Chain, error) {
	return LoadChainFromFileWithConsensus(
		path,
		LegacyOnlyPoWPolicy(),
		legacyProofVerifier{},
	)
}

// LoadChainFromFileWithConsensus loads and revalidates a stored chain using
// the supplied immutable policy and proof verifier.
func LoadChainFromFileWithConsensus(
	path string,
	policy PoWPolicy,
	verifier ProofVerifier,
) (*Chain, error) {
	if path == "" {
		return nil, fmt.Errorf(
			"storage path cannot be empty",
		)
	}

	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to read blockchain file: %w",
			err,
		)
	}

	var blocks []*Block

	if err := json.Unmarshal(
		data,
		&blocks,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode blockchain: %w",
			err,
		)
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf(
			"blockchain file contains no blocks",
		)
	}

	expectedGenesis := NewGenesisBlock()

	if blocks[0].Hash() != expectedGenesis.Hash() {
		return nil, fmt.Errorf(
			"invalid genesis block",
		)
	}

	chain, err := NewChainWithConsensus(policy, verifier)
	if err != nil {
		return nil, fmt.Errorf("invalid consensus policy: %w", err)
	}

	for i := 1; i < len(blocks); i++ {
		if err := chain.AddBlock(
			blocks[i],
		); err != nil {
			return nil, fmt.Errorf(
				"invalid stored block at height %d: %w",
				i,
				err,
			)
		}
	}

	return chain, nil
}
