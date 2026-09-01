package consensus

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
)

// BlockSubsidy preserves the current public-testnet subsidy behavior.
func BlockSubsidy(height uint64) uint64 {
	reward, _ := BlockSubsidyFor(params.MonetaryPolicyPublicTestnet, height)
	return reward
}

// BlockSubsidyFor returns the deterministic mining subsidy for a monetary
// policy and block height. Consensus arithmetic is integer base units only.
func BlockSubsidyFor(policy params.MonetaryPolicy, height uint64) (uint64, error) {
	if err := params.ValidateMonetaryPolicy(policy); err != nil {
		return 0, err
	}

	if policy == params.MonetaryPolicyPublicTestnet {
		halvings := height / params.HalvingInterval
		if halvings >= 64 {
			return 0, nil
		}
		return params.InitialBlockReward >> halvings, nil
	}

	if height == 0 || height > params.MainnetFinalSubsidyHeight {
		return 0, nil
	}

	epochIndex := (height - 1) / params.MainnetEpochLength
	if epochIndex >= uint64(len(params.MainnetEmissionEpochs)) {
		return 0, fmt.Errorf("mainnet epoch index %d out of range", epochIndex)
	}

	epoch := params.MainnetEmissionEpochs[epochIndex]
	base := epoch.Issuance / params.MainnetEpochLength
	remainder := epoch.Issuance % params.MainnetEpochLength
	offset := (height - 1) % params.MainnetEpochLength
	if offset < remainder {
		return base + 1, nil
	}
	return base, nil
}
