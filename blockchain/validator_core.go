package blockchain

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

// validateBlockCore is the legacy compatibility entry point used by tests and
// callers that do not carry a Chain. It remains Version-1/double-SHA256 only.
func validateBlockCore(block *Block, previous *Block) error {
	return validateBlockCoreWithProof(
		block,
		previous,
		LegacyOnlyPoWPolicy(),
		legacyProofVerifier{},
	)
}

// validateBlockCoreWithProof performs consensus checks under the immutable
// proof-version policy and verifier bound to a Chain.
func validateBlockCoreWithProof(
	block *Block,
	previous *Block,
	policy PoWPolicy,
	verifier ProofVerifier,
) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}
	if previous == nil {
		return fmt.Errorf("previous block cannot be nil")
	}
	if !policy.VersionAllowed(block.Version, block.Height) {
		expectedVersion, _ := policy.VersionAtHeight(block.Height)
		return fmt.Errorf(
			"invalid block version: expected %d at height %d, got %d",
			expectedVersion,
			block.Height,
			block.Version,
		)
	}
	if verifier == nil || !verifier.SupportsVersion(block.Version) {
		return fmt.Errorf("proof verifier does not support block version %d", block.Version)
	}

	expectedHeight := previous.Height + 1
	if block.Height != expectedHeight {
		return fmt.Errorf("invalid block height: expected %d, got %d", expectedHeight, block.Height)
	}
	if block.PreviousHash != previous.Hash() {
		return fmt.Errorf("invalid previous hash")
	}
	if block.Timestamp <= previous.Timestamp {
		return fmt.Errorf("invalid block timestamp: must be greater than previous block")
	}

	now := time.Now().Unix()
	if block.Timestamp > now+MaxFutureBlockSeconds {
		return fmt.Errorf("invalid block timestamp: too far in the future")
	}

	if block.MerkleRoot != block.CalculateMerkleRoot() {
		return fmt.Errorf("invalid merkle root")
	}
	if err := validateBlockTransactions(block); err != nil {
		return err
	}
	if !verifier.Verify(block) {
		return fmt.Errorf("invalid proof of work")
	}

	return nil
}

// validBlockProofOfWorkCore is the existing legacy Version-1 proof rule. Stage
// 1 keeps it unchanged; future Khushi Version-2 verification is injected via a
// different ProofVerifier rather than changing this compatibility function.
func validBlockProofOfWorkCore(block *Block) bool {
	hashBytes, err := hex.DecodeString(block.Hash())
	if err != nil {
		return false
	}

	hashInt := new(big.Int).SetBytes(hashBytes)
	target := blockTargetFromDifficulty(block.Difficulty)
	return target.Sign() > 0 && hashInt.Cmp(target) <= 0
}
