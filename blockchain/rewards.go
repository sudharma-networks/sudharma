package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

func CreditMinerReward(
	state *State,
	blockHeight uint64,
	minerAddress string,
	minerFees uint64,
) (uint64, error) {

	if state == nil {
		return 0, fmt.Errorf("state cannot be nil")
	}

	if minerAddress == "" {
		return 0, fmt.Errorf("miner address cannot be empty")
	}

	subsidy := consensus.BlockSubsidy(blockHeight)

	// Only the subsidy creates new SUDH.
	// Transaction fees are existing coins transferred from users.
	if subsidy > 0 {
		remaining := stateRemainingSupply(state)

		if subsidy > remaining {
			subsidy = remaining
		}

		if err := state.MintSupply(subsidy); err != nil {
			return 0, err
		}
	}

	if subsidy > ^uint64(0)-minerFees {
		return 0, fmt.Errorf("miner reward overflow")
	}

	totalReward := subsidy + minerFees

	if totalReward == 0 {
		return 0, nil
	}

	if err := state.Credit(
		minerAddress,
		totalReward,
	); err != nil {
		return 0, err
	}

	return totalReward, nil
}

func stateRemainingSupply(state *State) uint64 {
	issued := state.IssuedSupply()

	if issued >= params.MaxSupply {
		return 0
	}

	return params.MaxSupply - issued
}
