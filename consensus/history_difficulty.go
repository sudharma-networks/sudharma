package consensus

import (
	"sort"

	"github.com/sudharma-networks/sudharma/params"
)

const DifficultyHistoryWindow = 11

// MedianBlockInterval returns the median of recent completed
// block intervals. Non-positive intervals are ignored.
func MedianBlockInterval(intervals []int64) int64 {
	valid := make([]int64, 0, len(intervals))

	for _, interval := range intervals {
		if interval > 0 {
			valid = append(valid, interval)
		}
	}

	if len(valid) == 0 {
		return int64(params.TargetBlockTimeSeconds)
	}

	sort.Slice(valid, func(i, j int) bool {
		return valid[i] < valid[j]
	})

	return valid[len(valid)/2]
}

// NextDifficultyFromHistory calculates the next difficulty from
// recent completed block intervals instead of the candidate block
// timestamp itself. This reduces single-block timestamp influence.
func NextDifficultyFromHistory(currentDifficulty uint32, intervals []int64) uint32 {
	return NextDifficulty(currentDifficulty, MedianBlockInterval(intervals))
}
