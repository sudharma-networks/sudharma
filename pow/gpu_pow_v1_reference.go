package pow

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/sudharma-networks/sudharma/blockchain"
)

var gpuV1ReferenceHeaderDomain = []byte("SUDHARMA-GPU-POW-V1-REFERENCE-HEADER\x00")

// GPUV1ReferenceDigest computes the deterministic Khushi v1 CPU-reference
// digest for a canonical block-header prefix, nonce and height. It is a
// verification/reference boundary only; it does not provide a CPU production
// mining path. An empty verifier cache fails closed.
func GPUV1ReferenceDigest(headerPrefix []byte, nonce, height uint64, cache []GPUV1CacheNode) [32]byte {
	if len(cache) == 0 {
		return [32]byte{}
	}

	input := make([]byte, 0, len(gpuV1ReferenceHeaderDomain)+len(headerPrefix)+8)
	input = append(input, gpuV1ReferenceHeaderDomain...)
	input = append(input, headerPrefix...)

	var nonceBytes [8]byte
	binary.LittleEndian.PutUint64(nonceBytes[:], nonce)
	input = append(input, nonceBytes[:]...)

	headerDigest := sha256.Sum256(input)
	workSeed := binary.LittleEndian.Uint64(headerDigest[:8])
	programSeed := GPUV1ProgramSeed(GPUV1ProgramForHeight(height))
	mixDigest := gpuV1ProgrammaticGroupDigest(workSeed, programSeed, cache)
	return gpuV1FinalizeDigest(headerDigest, mixDigest)
}

// GPUV1HashBlockWithCache binds the existing canonical block serialization to
// the Khushi v1 reference digest. HeaderBytes(0) remains authoritative; only
// its final eight nonce bytes are replaced by the explicitly supplied nonce.
func GPUV1HashBlockWithCache(block *blockchain.Block, nonce uint64, cache []GPUV1CacheNode) string {
	if block == nil || len(cache) == 0 {
		return ""
	}
	header := block.HeaderBytes(0)
	if len(header) < 8 {
		return ""
	}
	digest := GPUV1ReferenceDigest(header[:len(header)-8], nonce, block.Height, cache)
	return hex.EncodeToString(digest[:])
}

// GPUV1CheckBlockWithCache verifies a Version-2 Khushi proof against the
// existing consensus target calculation. It fails closed for malformed input,
// non-Version-2 blocks, zero difficulty, and missing cache state.
func GPUV1CheckBlockWithCache(block *blockchain.Block, cache []GPUV1CacheNode) bool {
	if block == nil || block.Version != 2 || block.Difficulty == 0 || len(cache) == 0 {
		return false
	}
	hash := GPUV1HashBlockWithCache(block, block.Nonce, cache)
	return hash != "" && ValidHash(hash, block.Difficulty)
}
