package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/consensus"
)

// RecentBlockIntervals returns up to DifficultyHistoryWindow
// completed block intervals from the active chain.
func RecentBlockIntervals(chain *Chain) ([]int64, error) {
	if chain == nil {
		return nil, fmt.Errorf("chain cannot be nil")
	}

	chain.mu.RLock()
	defer chain.mu.RUnlock()

	return recentBlockIntervalsLocked(chain.blocks), nil
}

func recentBlockIntervalsLocked(blocks []*Block) []int64 {
	if len(blocks) < 2 {
		return nil
	}

	intervalCount := len(blocks) - 1
	if intervalCount > consensus.DifficultyHistoryWindow {
		intervalCount = consensus.DifficultyHistoryWindow
	}

	start := len(blocks) - intervalCount
	intervals := make([]int64, 0, intervalCount)

	for i := start; i < len(blocks); i++ {
		previous := blocks[i-1]
		current := blocks[i]
		if previous == nil || current == nil {
			continue
		}

		interval := current.Timestamp - previous.Timestamp
		if interval > 0 {
			intervals = append(intervals, interval)
		}
	}

	return intervals
}

// ExpectedNextDifficulty calculates the next consensus difficulty
// from completed chain history rather than the candidate timestamp.
func ExpectedNextDifficulty(chain *Chain) (uint32, error) {
	if chain == nil {
		return 0, fmt.Errorf("chain cannot be nil")
	}

	chain.mu.RLock()
	defer chain.mu.RUnlock()

	if len(chain.blocks) == 0 {
		return 0, fmt.Errorf("chain has no genesis block")
	}

	previous := chain.blocks[len(chain.blocks)-1]
	if previous == nil {
		return 0, fmt.Errorf("chain tip cannot be nil")
	}

	return consensus.NextDifficultyFromHistory(
		previous.Difficulty,
		recentBlockIntervalsLocked(chain.blocks),
	), nil
}
