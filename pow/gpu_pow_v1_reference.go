package pow

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/sudharma-networks/sudharma/blockchain"
)

var gpuV1ReferenceHeaderDomain = []byte("SUDHARMA-GPU-POW-V1-REFERENCE-HEADER\x00")

// gpuV1ReferenceDigest composes the complete deterministic CPU reference path
// for GPU-PoW v1 without activating it in consensus. The caller supplies the
// epoch cache so production cache sizing/lifecycle remains outside this pure
// reference primitive.
//
// Canonical reference encoding is:
//
//	domain || header-prefix || nonce-little-endian
//
// The resulting SHA-256 digest provides both the final header binding and the
// little-endian 64-bit work seed. The height selects the three-block
// programmatic schedule; the caller-selected cache binds the epoch dataset.
func gpuV1ReferenceDigest(header []byte, nonce, height uint64, cache []GPUV1CacheNode) [32]byte {
	input := make([]byte, 0, len(gpuV1ReferenceHeaderDomain)+len(header)+8)
	input = append(input, gpuV1ReferenceHeaderDomain...)
	input = append(input, header...)

	var nonceBytes [8]byte
	binary.LittleEndian.PutUint64(nonceBytes[:], nonce)
	input = append(input, nonceBytes[:]...)

	headerDigest := sha256.Sum256(input)
	workSeed := binary.LittleEndian.Uint64(headerDigest[:8])
	programSeed := GPUV1ProgramSeed(GPUV1ProgramForHeight(height))
	mix := gpuV1ProgrammaticGroupDigest(workSeed, programSeed, cache)

	return gpuV1FinalizeDigest(headerDigest, mix)
}

// gpuV1HashBlockWithCache composes the canonical block header prefix with the
// GPU-PoW v1 nonce encoding and returns the deterministic reference digest.
// The epoch cache is supplied explicitly so this primitive does not silently
// choose a consensus cache size before that parameter is frozen.
func gpuV1HashBlockWithCache(block *blockchain.Block, nonce uint64, cache []GPUV1CacheNode) string {
	if block == nil || len(cache) == 0 {
		return ""
	}

	headerWithNonce := block.HeaderBytes(0)
	if len(headerWithNonce) < 8 {
		return ""
	}
	headerPrefix := headerWithNonce[:len(headerWithNonce)-8]
	digest := gpuV1ReferenceDigest(headerPrefix, nonce, block.Height, cache)
	return hex.EncodeToString(digest[:])
}

// gpuV1CheckBlockWithCache verifies a Version-2 reference block against the
// existing Sudharma difficulty target while keeping cache selection explicit.
func gpuV1CheckBlockWithCache(block *blockchain.Block, cache []GPUV1CacheNode) bool {
	if block == nil || block.Version != 2 || len(cache) == 0 {
		return false
	}

	hash := gpuV1HashBlockWithCache(block, block.Nonce, cache)
	return hash != "" && ValidHash(hash, block.Difficulty)
}
