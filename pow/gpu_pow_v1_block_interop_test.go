package pow

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestGPUV1BlockInteroperabilityVectorProbe(t *testing.T) {
	block := &blockchain.Block{
		Version:      2,
		Height:       22501,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		Nonce:        0x0123456789abcdef,
		MinerAddress: "9ccdc094489874bed888ffe4bdf9b8298f4c5131",
	}
	cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(block.Height)), 8)
	header := block.HeaderBytes(0)
	headerPrefix := header[:len(header)-8]
	hash := gpuV1HashBlockWithCache(block, block.Nonce, cache)
	t.Fatalf("BLOCK_VECTOR header_prefix=%x nonce=%016x height=%d difficulty=%d hash=%s", headerPrefix, block.Nonce, block.Height, block.Difficulty, hash)
}
