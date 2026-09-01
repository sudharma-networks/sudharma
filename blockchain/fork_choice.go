package blockchain

import "fmt"

// BetterChain returns the chain that should be preferred.
//
// Sudharma Network fork-choice rule:
// 1. Chains from different networks are never comparable.
// 2. Prefer greater cumulative Proof-of-Work.
// 3. If total work is equal, prefer greater height.
// 4. If both are equal, keep the current chain.
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

	if current.Network() != candidate.Network() {
		return nil, fmt.Errorf(
			"cross-network fork choice rejected: current=%q candidate=%q",
			current.Network(),
			candidate.Network(),
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
