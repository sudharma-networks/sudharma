package miner

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
)

func MineNextBlock(
	chain *blockchain.Chain,
	state *blockchain.State,
	pool *mempool.Mempool,
	minerAddress string,
	maxAttempts uint64,
) (Result, uint64, error) {
	var emptyResult Result

	if chain == nil {
		return emptyResult, 0, fmt.Errorf("chain cannot be nil")
	}
	if state == nil {
		return emptyResult, 0, fmt.Errorf("state cannot be nil")
	}
	if pool == nil {
		return emptyResult, 0, fmt.Errorf("mempool cannot be nil")
	}
	if minerAddress == "" {
		return emptyResult, 0, fmt.Errorf("miner address cannot be empty")
	}
	if maxAttempts == 0 {
		return emptyResult, 0, fmt.Errorf("max mining attempts must be greater than zero")
	}

	previous := chain.Tip()
	if previous == nil {
		return emptyResult, 0, fmt.Errorf("chain tip cannot be nil")
	}

	block, err := blockchain.NewBlockFromMempoolWithPolicy(
		previous,
		pool,
		chain.PoWPolicy(),
	)
	if err != nil {
		return emptyResult, 0, fmt.Errorf("failed to build candidate block: %w", err)
	}
	if block.Version == 2 {
		return emptyResult, 0, fmt.Errorf("external GPU miner required")
	}
	if block.Timestamp <= previous.Timestamp {
		block.Timestamp = previous.Timestamp + 1
	}

	block.Difficulty, err = blockchain.ExpectedNextDifficulty(chain)
	if err != nil {
		return emptyResult, 0, fmt.Errorf("failed calculating mining difficulty: %w", err)
	}

	block.MinerAddress = minerAddress
	block.Nonce = 0
	block.UpdateMerkleRoot()

	result := Mine(block, 0, maxAttempts)
	if !result.Found {
		return result, 0, fmt.Errorf("no valid nonce found after %d attempts", maxAttempts)
	}

	if err := blockchain.ValidateBlockAgainstChain(chain, block); err != nil {
		return result, 0, fmt.Errorf("mined block failed validation: %w", err)
	}

	workingState := state.Clone()
	minerReward, err := blockchain.ProcessBlock(workingState, block, block.MinerAddress)
	if err != nil {
		return result, 0, fmt.Errorf("failed to process mined block: %w", err)
	}

	if err := chain.AddBlock(block); err != nil {
		return result, 0, fmt.Errorf("failed to add mined block to chain: %w", err)
	}
	if err := state.ReplaceWith(workingState); err != nil {
		return result, 0, fmt.Errorf("failed to commit blockchain state: %w", err)
	}

	for _, tx := range block.Transactions {
		if tx != nil {
			pool.RemoveTransaction(tx.ID)
		}
	}

	return result, minerReward, nil
}
