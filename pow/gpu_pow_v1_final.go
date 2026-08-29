package pow

import (
	"crypto/sha256"
	"encoding/binary"
)

var gpuV1FinalDigestDomain = []byte("SUDHARMA-GPU-POW-V1-FINAL\x00")

// gpuV1FinalizeDigest binds the canonical header digest to the reduced
// programmatic-mix digest. The encoding is fixed for cross-implementation
// vectors: 32 raw header-digest bytes followed by eight little-endian uint32
// mix words, all under a Sudharma-specific domain separator.
//
// This primitive is intentionally not wired into active Version-2 consensus
// yet. Task 5 will only replace the temporary HashBlockForVersion scaffold
// after the complete header+nonce reference path and interoperability vectors
// are fixed.
func gpuV1FinalizeDigest(headerDigest [32]byte, mix [8]uint32) [32]byte {
	input := make([]byte, 0, len(gpuV1FinalDigestDomain)+32+8*4)
	input = append(input, gpuV1FinalDigestDomain...)
	input = append(input, headerDigest[:]...)

	var word [4]byte
	for _, value := range mix {
		binary.LittleEndian.PutUint32(word[:], value)
		input = append(input, word[:]...)
	}

	return sha256.Sum256(input)
}
