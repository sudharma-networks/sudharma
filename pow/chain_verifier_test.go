package pow

import (
	"sync"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestChainProofVerifierUsesFrozenProductionCacheContract(t *testing.T) {
	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: 100}
	verifier, err := NewChainProofVerifier(policy)
	if err != nil {
		t.Fatal(err)
	}
	implementation, ok := verifier.(*chainProofVerifier)
	if !ok {
		t.Fatalf("verifier type = %T, want *chainProofVerifier", verifier)
	}
	if implementation.cacheNodeCount != GPUV1ProductionCacheNodes {
		t.Fatalf("production cache nodes = %d want %d", implementation.cacheNodeCount, GPUV1ProductionCacheNodes)
	}
	if !verifier.SupportsVersion(1) || !verifier.SupportsVersion(2) {
		t.Fatal("Khushi chain verifier must support Versions 1 and 2")
	}
	if verifier.SupportsVersion(3) {
		t.Fatal("Khushi chain verifier reported support for an unknown version")
	}
	if implementation.cachedEpochCount() != 0 {
		t.Fatal("production cache was built eagerly")
	}
}

func TestChainProofVerifierPolicyDispatchFailsClosedBeforeCacheWork(t *testing.T) {
	const activation uint64 = 5
	verifier, err := newChainProofVerifier(
		blockchain.PoWPolicy{GPUV1ActivationHeight: activation},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}

	if verifier.Verify(nil) {
		t.Fatal("nil block was accepted")
	}
	if verifier.Verify(&blockchain.Block{Version: 2, Height: activation - 1, Difficulty: 1}) {
		t.Fatal("Version 2 was accepted before activation")
	}
	if verifier.cachedEpochCount() != 0 {
		t.Fatal("pre-activation Version 2 rejection built a Khushi cache")
	}
	if !verifier.Verify(&blockchain.Block{Version: 1, Height: activation - 1, Difficulty: 1}) {
		t.Fatal("valid Version 1 proof was rejected before activation")
	}
	if verifier.Verify(&blockchain.Block{Version: 1, Height: activation, Difficulty: 1}) {
		t.Fatal("Version 1 was accepted at the Version-2 activation boundary")
	}
	if verifier.cachedEpochCount() != 0 {
		t.Fatal("Version 1 dispatch unexpectedly built a Khushi cache")
	}
	if !verifier.Verify(&blockchain.Block{Version: 2, Height: activation, Difficulty: 1}) {
		t.Fatal("valid Version 2 proof was rejected at activation")
	}
	if verifier.cachedEpochCount() != 1 {
		t.Fatal("Version 2 verification did not retain exactly one epoch cache")
	}
	if verifier.Verify(&blockchain.Block{Version: 3, Height: activation, Difficulty: 1}) {
		t.Fatal("unknown block version was accepted")
	}
	if verifier.Verify(&blockchain.Block{Version: 2, Height: activation, Difficulty: 0}) {
		t.Fatal("invalid Version 2 proof fell back to another proof path")
	}
}

func TestChainProofVerifierLegacyOnlyPolicyNeverBuildsKhushiCache(t *testing.T) {
	verifier, err := newChainProofVerifier(blockchain.LegacyOnlyPoWPolicy(), 8)
	if err != nil {
		t.Fatal(err)
	}
	if verifier.Verify(&blockchain.Block{Version: 2, Height: GPUV1EpochLength, Difficulty: 1}) {
		t.Fatal("legacy-only policy accepted Version 2")
	}
	if verifier.cachedEpochCount() != 0 {
		t.Fatal("legacy-only rejection built a Khushi cache")
	}
}

func TestChainProofVerifierRejectsEmptyTestCacheConfiguration(t *testing.T) {
	if _, err := newChainProofVerifier(
		blockchain.PoWPolicy{GPUV1ActivationHeight: 1},
		0,
	); err == nil {
		t.Fatal("zero-node verifier cache configuration was accepted")
	}
}

func TestChainProofVerifierReusesCacheWithinEpochAndReplacesAcrossEpochs(t *testing.T) {
	verifier, err := newChainProofVerifier(
		blockchain.PoWPolicy{GPUV1ActivationHeight: 1},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}

	first := verifier.cacheForHeight(1)
	sameEpoch := verifier.cacheForHeight(GPUV1EpochLength - 1)
	if len(first) != 8 || len(sameEpoch) != 8 {
		t.Fatalf("compact cache sizes = %d/%d want 8/8", len(first), len(sameEpoch))
	}
	if &first[0] != &sameEpoch[0] {
		t.Fatal("cache was rebuilt within the same epoch")
	}

	nextEpoch := verifier.cacheForHeight(GPUV1EpochLength)
	if len(nextEpoch) != 8 {
		t.Fatalf("next-epoch cache size = %d want 8", len(nextEpoch))
	}
	if &first[0] == &nextEpoch[0] {
		t.Fatal("cache backing storage was reused across epoch replacement")
	}
	if verifier.cachedEpochCount() != 1 {
		t.Fatal("verifier retained more than one epoch cache")
	}
}

func TestChainProofVerifierConcurrentVerificationAndCacheAccess(t *testing.T) {
	verifier, err := newChainProofVerifier(
		blockchain.PoWPolicy{GPUV1ActivationHeight: 1},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan string, 8)
	for i := 0; i < 8; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			height := uint64(1 + i)
			if i%2 == 1 {
				height += GPUV1EpochLength
			}
			if i%3 == 0 {
				cache := verifier.cacheForHeight(height)
				if len(cache) != 8 {
					errs <- "concurrent cache access returned wrong cache size"
				}
				return
			}
			if !verifier.Verify(&blockchain.Block{Version: 2, Height: height, Difficulty: 1}) {
				errs <- "concurrent valid Version 2 proof was rejected"
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if verifier.cachedEpochCount() != 1 {
		t.Fatal("concurrent verification retained more than one epoch cache")
	}
}
