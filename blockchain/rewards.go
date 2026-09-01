package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

// CreditMinerReward credits the public-testnet miner reward for a mined block.
//
// Total miner reward =
//
//	block subsidy + miner transaction fees
//
// This wrapper preserves the existing public-testnet call sites.
func CreditMinerReward(
	state *State,
	blockHeight uint64,
	minerAddress string,
	minerFees uint64,
) (uint64, error) {
	return CreditMinerRewardFor(
		state,
		params.MonetaryPolicyPublicTestnet,
		blockHeight,
		minerAddress,
		minerFees,
	)
}

// CreditMinerRewardFor credits the miner reward for a mined block under an
// explicit monetary policy.
//
// Total miner reward =
//
//	block subsidy + miner transaction fees
//
// The block subsidy is newly issued supply under the selected policy.
// Miner transaction fees are already existing circulating supply.
func CreditMinerRewardFor(
	state *State,
	policy params.MonetaryPolicy,
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

	subsidy, err := consensus.BlockSubsidyFor(policy, blockHeight)
	if err != nil {
		return 0, err
	}

	// Only the subsidy creates new SUDH.
	// Transaction fees are existing coins transferred from users.
	if subsidy > 0 {
		remaining := stateRemainingSupplyFor(state, policy)

		if subsidy > remaining {
			subsidy = remaining
		}

		if err := state.MintSupplyFor(policy, subsidy); err != nil {
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

	if err := state.Credit(minerAddress, totalReward); err != nil {
		return 0, err
	}

	return totalReward, nil
}

func stateRemainingSupplyFor(
	state *State,
	policy params.MonetaryPolicy,
) uint64 {
	maxSupply := params.MaxSupplyFor(policy)
	issued := state.IssuedSupply()

	if issued >= maxSupply {
		return 0
	}

	return maxSupply - issued
}
