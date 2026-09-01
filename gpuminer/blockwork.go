package gpuminer

import (
	"encoding/hex"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

// WorkFromCandidateBlock builds Khushi hasher inputs from a Sudharma candidate block.
func WorkFromCandidateBlock(block *blockchain.Block, poolDifficulty uint32, poolTarget string) Work {
	if block == nil {
		return Work{}
	}
	return Work{
		Algorithm:     params.ProductionMiningAlgorithm,
		Version:       2,
		Height:        block.Height,
		Difficulty:    poolDifficulty,
		Target:        poolTarget,
		HeaderPrefix:  hex.EncodeToString(block.HeaderBytes(0)),
		RewardAddress: block.MinerAddress,
		Block:         block,
	}
}
