package blockchain

import (
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
)

// PoWPolicy selects the only block version permitted at a given height.
// A policy is copied into a Chain and must not change during its lifetime.
type PoWPolicy struct {
	GPUV1ActivationHeight uint64
}

// LegacyOnlyPoWPolicy returns the safe disabled-by-default consensus policy.
func LegacyOnlyPoWPolicy() PoWPolicy {
	return PoWPolicy{GPUV1ActivationHeight: params.GPUV1ActivationDisabled}
}

// PoWPolicyForNetwork returns the frozen proof-version policy for a network.
// Both networks remain disabled until a later dedicated activation change.
func PoWPolicyForNetwork(network params.NetworkID) (PoWPolicy, error) {
	switch network {
	case params.NetworkPublicTestnet:
		return PoWPolicy{GPUV1ActivationHeight: params.GPUV1TestnetActivationHeight}, nil
	case params.NetworkMainnet:
		return PoWPolicy{GPUV1ActivationHeight: params.GPUV1MainnetActivationHeight}, nil
	default:
		return PoWPolicy{}, fmt.Errorf("unknown network %q", network)
	}
}

// VersionAllowed reports whether version is the only version selected for
// height. Unknown and future versions are rejected explicitly.
func (p PoWPolicy) VersionAllowed(version uint32, height uint64) bool {
	if version != 1 && version != 2 {
		return false
	}
	if p.GPUV1ActivationHeight == params.GPUV1ActivationDisabled ||
		height < p.GPUV1ActivationHeight {
		return version == 1
	}
	return version == 2
}

// VersionAtHeight returns the single block version selected at height.
func (p PoWPolicy) VersionAtHeight(height uint64) (uint32, error) {
	if p.GPUV1ActivationHeight == params.GPUV1ActivationDisabled ||
		height < p.GPUV1ActivationHeight {
		return 1, nil
	}
	return 2, nil
}
