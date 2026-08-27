package pow

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
)

const gpuPoWV1Domain = "SUDHARMA-GPU-POW-V1\x00"

// HashBlock calculates the canonical legacy Sudharma Network PoW hash.
func HashBlock(block *blockchain.Block, nonce uint64) string {
	header := block.HeaderBytes(nonce)

	first := sha256Hash(header)
	second := sha256Hash(first)

	return hex.EncodeToString(second)
}

// HashBlockForVersion dispatches proof-of-work hashing by block version.
// Version 1 remains byte-for-byte compatible with the legacy chain. Version 2
// is explicitly domain-separated for GPU-PoW v1. The current v2 digest is the
// minimal deterministic consensus scaffold; the memory-hard/programmatic mix
// will replace gpuPoWV1Digest behind fixed cross-implementation test vectors.
func HashBlockForVersion(block *blockchain.Block, nonce uint64) string {
	if block.Version < 2 {
		return HashBlock(block, nonce)
	}

	return hex.EncodeToString(gpuPoWV1Digest(block.HeaderBytes(nonce)))
}

func gpuPoWV1Digest(header []byte) []byte {
	input := make([]byte, 0, len(gpuPoWV1Domain)+len(header))
	input = append(input, gpuPoWV1Domain...)
	input = append(input, header...)

	first := sha256Hash(input)
	second := sha256Hash(first)
	return second
}

// sha256Hash returns a SHA-256 digest.
func sha256Hash(data []byte) []byte {
	hash := make([]byte, 32)

	sum := sha256.Sum256(data)
	copy(hash, sum[:])

	return hash
}

// TargetFromDifficulty returns the canonical consensus PoW target.
func TargetFromDifficulty(difficulty uint32) *big.Int {
	return consensus.TargetFromDifficulty(
		difficulty,
	)
}

// ValidHash checks whether a hash satisfies the PoW target.
func ValidHash(hash string, difficulty uint32) bool {
	if difficulty == 0 {
		return false
	}
	target := TargetFromDifficulty(difficulty)

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}

	hashInt := new(big.Int).SetBytes(hashBytes)

	return hashInt.Cmp(target) <= 0
}

// CheckBlock verifies the block's proof of work using its declared version.
func CheckBlock(block *blockchain.Block) bool {
	hash := HashBlockForVersion(block, block.Nonce)

	return ValidHash(hash, block.Difficulty)
}
