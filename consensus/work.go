package consensus

import "math/big"

// TargetFromDifficulty converts difficulty into the canonical 256-bit PoW target.
func TargetFromDifficulty(difficulty uint32) *big.Int {
	if difficulty == 0 {
		return big.NewInt(0)
	}

	maxHash := new(big.Int).Lsh(big.NewInt(1), 256)
	maxHash.Sub(maxHash, big.NewInt(1))

	return new(big.Int).Div(
		maxHash,
		new(big.Int).SetUint64(uint64(difficulty)),
	)
}

// WorkFromDifficulty returns exact target-derived cumulative chain work.
// Work = floor(2^256 / (target + 1)).
func WorkFromDifficulty(difficulty uint32) *big.Int {
	if difficulty == 0 {
		return big.NewInt(0)
	}

	target := TargetFromDifficulty(difficulty)
	if target.Sign() <= 0 {
		return big.NewInt(0)
	}

	twoTo256 := new(big.Int).Lsh(big.NewInt(1), 256)
	denominator := new(big.Int).Add(target, big.NewInt(1))

	return new(big.Int).Div(twoTo256, denominator)
}
