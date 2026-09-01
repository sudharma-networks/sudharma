package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sudharma-networks/sudharma/params"
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

// LoadChainFromFile loads a public-testnet chain from disk for compatibility.
func LoadChainFromFile(path string) (*Chain, error) {
	return LoadChainFromFileFor(path, params.NetworkPublicTestnet)
}

// LoadChainFromFileFor loads and validates a Sudharma Network chain from disk
// under an explicit network identity. It does not authorize runtime mainnet;
// callers must obtain their active network through the launch-gated parser.
func LoadChainFromFileFor(path string, network params.NetworkID) (*Chain, error) {
	if path == "" {
		return nil, fmt.Errorf(
			"storage path cannot be empty",
		)
	}
	if _, err := params.MonetaryPolicyFor(network); err != nil {
		return nil, err
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
	if blocks[0] == nil {
		return nil, fmt.Errorf("invalid genesis block")
	}

	chain, err := newChainFromGenesisForNetwork(network, blocks[0])
	if err != nil {
		return nil, fmt.Errorf("invalid genesis block: %w", err)
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
