package consensus

import "github.com/sudharma-networks/sudharma/params"

// BlockSubsidy returns the mining subsidy for a given block height.
func BlockSubsidy(height uint64) uint64 {
	halvings := height / params.HalvingInterval

	// uint64 cannot safely be shifted by 64 or more
	// for our monetary calculation.
	if halvings >= 64 {
		return 0
	}

	reward := params.InitialBlockReward >> halvings

	return reward
}
