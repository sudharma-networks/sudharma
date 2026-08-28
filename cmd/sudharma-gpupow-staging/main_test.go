package main

import (
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
