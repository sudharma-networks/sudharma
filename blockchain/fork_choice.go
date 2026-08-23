package blockchain

import "fmt"

// BetterChain returns the chain that should be preferred.
//
// Sudharma Network fork-choice rule:
// 1. Prefer greater cumulative Proof-of-Work.
// 2. If total work is equal, prefer greater height.
// 3. If both are equal, keep the current chain.
func BetterChain(
	current *Chain,
	candidate *Chain,
) (*Chain, error) {

	if current == nil {
		return nil, fmt.Errorf(
			"current chain cannot be nil",
		)
	}

	if candidate == nil {
		return nil, fmt.Errorf(
			"candidate chain cannot be nil",
		)
	}

	currentWork :=
		current.TotalWork()

	candidateWork :=
		candidate.TotalWork()

	// Candidate has more cumulative work.
	if candidateWork.Cmp(currentWork) > 0 {
		return candidate, nil
	}

	// Current has more cumulative work.
	if candidateWork.Cmp(currentWork) < 0 {
		return current, nil
	}

	// Equal work: use height as secondary rule.
	if candidate.Height() > current.Height() {
		return candidate, nil
	}

	// Equal work and equal/lower height:
	// keep current chain to avoid unnecessary switching.
	return current, nil
}
