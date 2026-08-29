package rpc

import (
	"math"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

type gpuActivationStatus struct {
	Phase            string
	ActivationHeight *uint64
	NextBlockVersion uint32
}

func deriveGPUActivationStatus(policy blockchain.PoWPolicy, tipHeight uint64) gpuActivationStatus {
	status := gpuActivationStatus{
		Phase:            "disabled",
		NextBlockVersion: 1,
	}
	if policy.GPUV1ActivationHeight == params.GPUV1ActivationDisabled {
		return status
	}

	height := policy.GPUV1ActivationHeight
	status.ActivationHeight = &height
	if tipHeight >= height {
		status.Phase = "active"
		status.NextBlockVersion = 2
		return status
	}

	status.Phase = "armed"
	if tipHeight != math.MaxUint64 && tipHeight+1 >= height {
		status.NextBlockVersion = 2
	}
	return status
}
