package main

import (
	"bytes"
	"encoding/hex"
	"testing"

	gpupowv1 "github.com/sudharma-networks/sudharma/compatibility/gpupowv1"
	"github.com/sudharma-networks/sudharma/pow"
	"github.com/sudharma-networks/sudharma/rpc"
)

func TestVerifyStagingSolutionUsesIndependentDigest(t *testing.T) {
	header := []byte("physical-gpu-staging-gate")
	target := make([]byte, 32)
	target[0] = 0x0f
	for i := 1; i < len(target); i++ {
		target[i] = 0xff
	}
	challenge := rpc.MiningStagingChallenge{
		ChallengeID:     "test",
		Algorithm:       pow.GPUV1AlgorithmID,
		Height:          0,
		HeaderPrefixHex: hex.EncodeToString(header),
		TargetHex:       hex.EncodeToString(target),
		CacheNodes:      8,
		Staging:         true,
	}

	var nonce uint64
	for ; nonce < 4096; nonce++ {
		digest := gpupowv1.Digest(header, nonce, 0, 8)
		if digestAtOrBelowTarget(digest, target) {
			break
		}
	}
	if nonce == 4096 {
		t.Fatal("test target unexpectedly had no solution")
	}
	if !verifyStagingSolution(challenge, nonce) {
		t.Fatal("independent staging verifier rejected known valid nonce")
	}

	mutated := challenge
	mutated.CacheNodes = 9
	if verifyStagingSolution(mutated, nonce) {
		t.Fatal("staging verifier accepted unsupported cache-node mutation")
	}
	mutated = challenge
	mutated.Staging = false
	if verifyStagingSolution(mutated, nonce) {
		t.Fatal("staging verifier accepted a non-staging challenge")
	}
}

func TestStagingChallengeProviderProducesFreshChallenge(t *testing.T) {
	first, firstHeight, firstCacheNodes, firstTarget, err := stagingChallengeProvider()
	if err != nil {
		t.Fatal(err)
	}
	second, secondHeight, secondCacheNodes, secondTarget, err := stagingChallengeProvider()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 || len(second) != 32 {
		t.Fatalf("staging challenge header lengths: got %d and %d want 32", len(first), len(second))
	}
	if bytes.Equal(first, second) {
		t.Fatal("consecutive staging challenges must not reuse the same header")
	}
	if firstHeight != stagingHeight || secondHeight != stagingHeight || firstCacheNodes != stagingCacheNodes || secondCacheNodes != stagingCacheNodes {
		t.Fatal("fresh challenges must preserve the constrained staging height/cache contract")
	}
	if !bytes.Equal(firstTarget, secondTarget) || len(firstTarget) != 32 {
		t.Fatal("fresh challenges must preserve the staging target")
	}
}
