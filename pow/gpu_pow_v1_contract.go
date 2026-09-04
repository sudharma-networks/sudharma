package pow

import (
	"crypto/sha256"
	"encoding/binary"
)

const (
	// GPUV1AlgorithmID is the consensus-visible identifier for Sudharma's
	// first GPU-oriented proof-of-work algorithm.
	GPUV1AlgorithmID = "sudharma-gpupow-v1"

	// GPUV1EpochLength changes the deterministic verifier cache epoch every
	// 7,500 blocks. This is part of the Khushi v1 consensus contract.
	GPUV1EpochLength uint64 = 7500

	// GPUV1CacheNodeBytes is the fixed cache-node width used by the reference
	// verifier and cross-implementation vectors.
	GPUV1CacheNodeBytes uint64 = 64

	// GPUV1ProductionCacheBytes is the cache memory committed to the production
	// CPU reference verifier. Hardware-only dataset/reserve/VRAM policy remains
	// outside this consensus package.
	GPUV1ProductionCacheBytes uint64 = 16 << 20

	// GPUV1ProductionCacheNodes is derived from the frozen byte/node contract.
	GPUV1ProductionCacheNodes uint32 = 262144
)

var gpuV1EpochSeedDomain = []byte("SUDHARMA-GPU-POW-V1-EPOCH-SEED\x00")

// GPUV1EpochForHeight returns the deterministic cache epoch for a block height.
func GPUV1EpochForHeight(height uint64) uint64 {
	return height / GPUV1EpochLength
}

// GPUV1EpochSeed derives the Sudharma-domain-separated seed for an epoch.
func GPUV1EpochSeed(epoch uint64) [32]byte {
	input := make([]byte, len(gpuV1EpochSeedDomain)+8)
	copy(input, gpuV1EpochSeedDomain)
	binary.BigEndian.PutUint64(input[len(gpuV1EpochSeedDomain):], epoch)
	return sha256.Sum256(input)
}
