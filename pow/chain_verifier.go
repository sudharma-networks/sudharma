package pow

import (
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// chainProofVerifier is the consensus-facing Khushi verifier. It supports the
// legacy Version-1 proof and the Version-2 Khushi reference verifier, while the
// immutable PoW policy selects exactly one admissible version at each height.
//
// The CPU reference path is verification-only. This type does not expose a CPU
// mining API or fallback path.
type chainProofVerifier struct {
	policy         blockchain.PoWPolicy
	cacheNodeCount uint32

	mu         sync.Mutex
	cacheSet   bool
	cacheEpoch uint64
	cache      []GPUV1CacheNode
}

// NewChainProofVerifier returns a verifier with the frozen production Khushi
// cache size. Cache construction remains lazy so a legacy-only or
// pre-activation chain pays no Version-2 cache cost.
func NewChainProofVerifier(policy blockchain.PoWPolicy) (blockchain.ProofVerifier, error) {
	return newChainProofVerifier(policy, GPUV1ProductionCacheNodes)
}

// newChainProofVerifier permits compact deterministic caches in tests without
// changing the production consensus cache contract.
func newChainProofVerifier(policy blockchain.PoWPolicy, cacheNodeCount uint32) (*chainProofVerifier, error) {
	if cacheNodeCount == 0 {
		return nil, fmt.Errorf("Khushi verifier cache node count must be positive")
	}
	return &chainProofVerifier{
		policy:         policy,
		cacheNodeCount: cacheNodeCount,
	}, nil
}

func (v *chainProofVerifier) SupportsVersion(version uint32) bool {
	return v != nil && (version == 1 || version == 2)
}

// Verify applies the immutable version policy before dispatching to either
// proof implementation. In particular, disallowed Version-2 blocks are
// rejected before any cache allocation or dataset work.
func (v *chainProofVerifier) Verify(block *blockchain.Block) bool {
	if v == nil || block == nil || !v.policy.VersionAllowed(block.Version, block.Height) {
		return false
	}

	switch block.Version {
	case 1:
		return CheckBlock(block)
	case 2:
		cache := v.cacheForHeight(block.Height)
		return GPUV1CheckBlockWithCache(block, cache)
	default:
		return false
	}
}

// cacheForHeight returns the single retained epoch cache, replacing it only
// when verification crosses an epoch boundary. Building while holding the
// mutex avoids duplicate production-size allocations under concurrent first
// use. A returned slice is immutable after construction, so replacing the
// retained slice does not mutate a cache already in use by another verifier.
func (v *chainProofVerifier) cacheForHeight(height uint64) []GPUV1CacheNode {
	if v == nil || v.cacheNodeCount == 0 {
		return nil
	}
	epoch := GPUV1EpochForHeight(height)

	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.cacheSet || v.cacheEpoch != epoch {
		v.cache = GPUV1BuildCache(GPUV1EpochSeed(epoch), v.cacheNodeCount)
		if len(v.cache) == 0 {
			v.cacheSet = false
			return nil
		}
		v.cacheEpoch = epoch
		v.cacheSet = true
	}
	return v.cache
}

func (v *chainProofVerifier) cachedEpochCount() int {
	if v == nil {
		return 0
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.cacheSet {
		return 0
	}
	return 1
}
