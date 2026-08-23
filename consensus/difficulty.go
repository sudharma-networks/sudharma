package consensus

import (
	"github.com/sudharma-networks/sudharma/params"
)

// NextDifficulty calculates the next block difficulty.
//
// Development rule:
//   - Target = 60 seconds
//   - If actual block time is less than half target,
//     difficulty increases by 25%
//   - If actual block time is more than double target,
//     difficulty decreases by 20%
//   - Otherwise difficulty remains unchanged
//
// Minimum difficulty is always 1.
func NextDifficulty(
	currentDifficulty uint32,
	actualBlockTimeSeconds int64,
) uint32 {

	if currentDifficulty == 0 {
		currentDifficulty = 1
	}

	target := int64(params.TargetBlockTimeSeconds)

	// Very fast blocks.
	if actualBlockTimeSeconds < target/2 {
		increase := currentDifficulty / 4

		if increase == 0 {
			increase = 1
		}

		// Protect uint32 overflow.
		if currentDifficulty > ^uint32(0)-increase {
			return ^uint32(0)
		}

		return currentDifficulty + increase
	}

	// Very slow blocks.
	if actualBlockTimeSeconds > target*2 {
		decrease := currentDifficulty / 5

		if decrease == 0 {
			decrease = 1
		}

		if decrease >= currentDifficulty {
			return 1
		}

		next := currentDifficulty - decrease

		if next < 1 {
			return 1
		}

		return next
	}

	return currentDifficulty
}
