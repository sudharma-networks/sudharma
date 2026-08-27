package pow

import (
	"crypto/sha256"
	"encoding/binary"
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
