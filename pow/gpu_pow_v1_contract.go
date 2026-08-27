package pow

import (
	"crypto/sha256"
	"encoding/binary"
)

const (
	// GPUV1AlgorithmID is the consensus-visible identifier for Sudharma's
	// first GPU-oriented proof-of-work algorithm.
	GPUV1AlgorithmID = "sudharma-gpupow-v1"

	// GPUV1EpochLength changes the cache/DAG epoch approximately every 5.2
	// days at Sudharma's 60-second target block interval.
	GPUV1EpochLength uint64 = 7500
)

var gpuV1EpochSeedDomain = []byte("SUDHARMA-GPU-POW-V1-EPOCH-SEED\x00")

// GPUV1EpochForHeight returns the deterministic cache/DAG epoch for a height.
func GPUV1EpochForHeight(height uint64) uint64 {
	return height / GPUV1EpochLength
}

// GPUV1EpochSeed derives a Sudharma-domain-separated seed for an epoch. This
// seed is the root input for later cache/DAG generation and is intentionally
// independent from other chains' Ethash/ProgPoW epoch seeds.
func GPUV1EpochSeed(epoch uint64) [32]byte {
	input := make([]byte, len(gpuV1EpochSeedDomain)+8)
	copy(input, gpuV1EpochSeedDomain)
	binary.BigEndian.PutUint64(input[len(gpuV1EpochSeedDomain):], epoch)
	return sha256.Sum256(input)
}
