package pow

import (
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/compatibility/gpupowv1"
)

var productionGPUV1CacheNodeCount = uint32(
	gpupowv1.GPUV1ProductionMemory.CacheBytes /
		gpupowv1.GPUV1ProductionMemory.ItemBytes,
)

type chainProofVerifier struct {
	policy      blockchain.PoWPolicy
	mu          sync.Mutex
	cachedEpoch uint64
	cache       []GPUV1CacheNode
}

// NewChainProofVerifier returns a CPU verifier for legacy and GPU-PoW blocks.
// It verifies Version 2 but does not provide a CPU production-mining path.
func NewChainProofVerifier(policy blockchain.PoWPolicy) (blockchain.ProofVerifier, error) {
	if err := gpupowv1.GPUV1ProductionMemory.Validate(); err != nil {
		return nil, fmt.Errorf("invalid GPU-PoW production memory policy: %w", err)
	}
	if productionGPUV1CacheNodeCount == 0 {
		return nil, fmt.Errorf("GPU-PoW production cache cannot be empty")
	}
	return &chainProofVerifier{policy: policy}, nil
}

func (v *chainProofVerifier) SupportsVersion(version uint32) bool {
	return v != nil && (version == 1 || version == 2)
}

func (v *chainProofVerifier) Verify(block *blockchain.Block) bool {
	if v == nil || block == nil || !v.policy.VersionAllowed(block.Version, block.Height) {
		return false
	}
	switch block.Version {
	case 1:
		return CheckBlock(block)
	case 2:
		return CheckBlockWithCache(block, v.cacheForHeight(block.Height))
	default:
		return false
	}
}

func (v *chainProofVerifier) cacheForHeight(height uint64) []GPUV1CacheNode {
	epoch := GPUV1EpochForHeight(height)
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.cache) == 0 || v.cachedEpoch != epoch {
		v.cache = GPUV1BuildCache(
			GPUV1EpochSeed(epoch),
			productionGPUV1CacheNodeCount,
		)
		v.cachedEpoch = epoch
	}
	return v.cache
}

func (v *chainProofVerifier) cachedEpochCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.cache) == 0 {
		return 0
	}
	return 1
}
