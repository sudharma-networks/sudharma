package pow

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
)

// HashBlock calculates the canonical legacy Sudharma Network PoW hash.
func HashBlock(block *blockchain.Block, nonce uint64) string {
	header := block.HeaderBytes(nonce)

	first := sha256Hash(header)
	second := sha256Hash(first)

	return hex.EncodeToString(second)
}

// HashBlockForVersion dispatches proof-of-work hashing where no external
// version-specific context is required. Legacy Version-1 blocks remain
// byte-for-byte compatible. Version 2 intentionally returns no hash here:
// GPU-PoW v1 requires an explicit epoch cache until production cache sizing
// and lifecycle are frozen by the activation task.
func HashBlockForVersion(block *blockchain.Block, nonce uint64) string {
	if block == nil {
		return ""
	}
	if block.Version < 2 {
		return HashBlock(block, nonce)
	}
	return ""
}

// HashBlockForVersionWithCache dispatches proof-of-work hashing with explicit
// GPU-PoW v1 cache context. This is the canonical Version-2 reference path used
// by interoperability and pre-activation validation tests; it does not choose
// a production cache size implicitly.
func HashBlockForVersionWithCache(block *blockchain.Block, nonce uint64, cache []GPUV1CacheNode) string {
	if block == nil {
		return ""
	}
	if block.Version < 2 {
		return HashBlock(block, nonce)
	}
	if block.Version != 2 {
		return ""
	}
	return gpuV1HashBlockWithCache(block, nonce, cache)
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

// CheckBlock verifies proof of work only for versions that require no external
// validation context. Version 2 remains disabled here until activation rules
// provide the frozen GPU-PoW cache policy.
func CheckBlock(block *blockchain.Block) bool {
	if block == nil {
		return false
	}
	hash := HashBlockForVersion(block, block.Nonce)
	return hash != "" && ValidHash(hash, block.Difficulty)
}

// CheckBlockWithCache verifies proof of work with explicit Version-2 cache
// context while preserving legacy Version-1 validation semantics.
func CheckBlockWithCache(block *blockchain.Block, cache []GPUV1CacheNode) bool {
	if block == nil {
		return false
	}
	hash := HashBlockForVersionWithCache(block, block.Nonce, cache)
	return hash != "" && ValidHash(hash, block.Difficulty)
}
