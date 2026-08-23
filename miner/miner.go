package miner

import (
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/pow"
)

// Result contains the result of a mining attempt.
type Result struct {
	Block     *blockchain.Block
	Hash      string
	Nonce     uint64
	Duration  time.Duration
	HashesRun uint64
	Found     bool
}

// Mine searches for a valid nonce.
func Mine(block *blockchain.Block, startNonce uint64, maxAttempts uint64) Result {
	start := time.Now()

	for i := uint64(0); i < maxAttempts; i++ {
		nonce := startNonce + i

		hash := pow.HashBlock(block, nonce)

		if pow.ValidHash(hash, block.Difficulty) {
			block.Nonce = nonce

			return Result{
				Block:     block,
				Hash:      hash,
				Nonce:     nonce,
				Duration:  time.Since(start),
				HashesRun: i + 1,
				Found:     true,
			}
		}
	}

	return Result{
		Block:     block,
		Duration:  time.Since(start),
		HashesRun: maxAttempts,
		Found:     false,
	}
}
