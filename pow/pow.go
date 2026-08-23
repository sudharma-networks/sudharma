package pow

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/consensus"
)

// HashBlock calculates the canonical Sudharma Network PoW hash.
func HashBlock(block *blockchain.Block, nonce uint64) string {
	header := block.HeaderBytes(nonce)

	first := sha256Hash(header)
	second := sha256Hash(first)

	return hex.EncodeToString(second)
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

// CheckBlock verifies the block's proof of work.
func CheckBlock(block *blockchain.Block) bool {
	hash := HashBlock(block, block.Nonce)

	return ValidHash(hash, block.Difficulty)
}
