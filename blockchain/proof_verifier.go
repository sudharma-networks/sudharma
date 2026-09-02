package blockchain

// ProofVerifier declares the block versions it can verify and validates the
// proof selected by a chain's immutable PoW policy. Stage 1 intentionally
// provides only the legacy Version-1 verifier; Khushi Version-2 verification
// is a later reconciliation stage and remains unusable until then.
type ProofVerifier interface {
	SupportsVersion(version uint32) bool
	Verify(block *Block) bool
}

type legacyProofVerifier struct{}

func (legacyProofVerifier) SupportsVersion(version uint32) bool {
	return version == 1
}

func (legacyProofVerifier) Verify(block *Block) bool {
	return block != nil && block.Version == 1 && validBlockProofOfWorkCore(block)
}
