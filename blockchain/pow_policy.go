package blockchain

import "github.com/sudharma-networks/sudharma/params"

// PoWPolicy selects the only block version permitted at a given height.
// A policy is copied into a Chain and must not change during its lifetime.
type PoWPolicy struct {
	GPUV1ActivationHeight uint64
}

// LegacyOnlyPoWPolicy returns the safe, disabled-by-default consensus policy.
func LegacyOnlyPoWPolicy() PoWPolicy {
	return PoWPolicy{GPUV1ActivationHeight: params.GPUV1ActivationDisabled}
}

// VersionAllowed reports whether version is the only version selected for
// height. Unknown and future versions are rejected explicitly.
func (p PoWPolicy) VersionAllowed(version uint32, height uint64) bool {
	if p.GPUV1ActivationHeight == params.GPUV1ActivationDisabled ||
		height < p.GPUV1ActivationHeight {
		return version == 1
	}
	return version == 2
}

// VersionAtHeight returns the block version selected for height.
func (p PoWPolicy) VersionAtHeight(height uint64) (uint32, error) {
	if p.GPUV1ActivationHeight == params.GPUV1ActivationDisabled ||
		height < p.GPUV1ActivationHeight {
		return 1, nil
	}
	return 2, nil
}
