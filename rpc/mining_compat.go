package rpc

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
)

// POWCompatWork exposes Sudharma GPU mining work using field names familiar from
// Bitcoin/Ravencoin getblocktemplate and Ethereum eth_getWork. Integrators can
// map these aliases without adopting legacy JSON-RPC method names on sudharma-rpcd.
type POWCompatWork struct {
	GetBlockTemplate map[string]any `json:"getblocktemplate"`
	EthGetWork       map[string]any `json:"eth_getWork"`
}

func buildPOWCompatWork(block *blockchain.Block, parent string, blockReward uint64) POWCompatWork {
	target := consensus.TargetFromDifficulty(block.Difficulty).Text(16)
	headerHash := block.Hash()
	return POWCompatWork{
		GetBlockTemplate: map[string]any{
			"previousblockhash":          parent,
			"height":                     block.Height,
			"target":                     target,
			"bits":                       fmt.Sprintf("%08x", block.Difficulty),
			"curtime":                    block.Timestamp,
			"mintime":                    block.Timestamp,
			"coinbasevalue":              blockReward,
			"default_witness_commitment": "",
			"transactions":               len(block.Transactions),
		},
		EthGetWork: map[string]any{
			"header_hash": headerHash,
			"seed_hash":   headerHash,
			"boundary":    target,
			"height":      block.Height,
		},
	}
}
