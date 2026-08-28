package rpc

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/sudharma-networks/sudharma/pow"
)

func TestMiningStagingChallengeIsExplicitlyNonConsensus(t *testing.T) {
	service := NewMiningStagingService(func(challenge MiningStagingChallenge, nonce uint64) bool {
		return nonce == 7
	})

	challenge, err := service.Issue([]byte("hardware-gate"), 0, 8, bytes.Repeat([]byte{0xff}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if !challenge.Staging {
		t.Fatal("staging challenge must identify itself as non-consensus staging work")
	}
	if challenge.Algorithm != pow.GPUV1AlgorithmID {
		t.Fatalf("algorithm mismatch: got %q want %q", challenge.Algorithm, pow.GPUV1AlgorithmID)
	}
	if challenge.CacheNodes != 8 {
		t.Fatalf("cache node count mismatch: got %d want 8", challenge.CacheNodes)
	}
	if challenge.ChallengeID == "" || challenge.HeaderPrefixHex != hex.EncodeToString([]byte("hardware-gate")) {
		t.Fatal("staging challenge must bind a stable ID to the exact header prefix")
	}
}

func TestMiningStagingSubmitRejectsMutationAndUsesVerifier(t *testing.T) {
	verified := 0
	service := NewMiningStagingService(func(challenge MiningStagingChallenge, nonce uint64) bool {
		verified++
		return nonce == 7
	})
	challenge, err := service.Issue([]byte("hardware-gate"), 0, 8, bytes.Repeat([]byte{0xff}, 32))
	if err != nil {
		t.Fatal(err)
	}

	mutated := MiningStagingSolution{Challenge: challenge, Nonce: 7}
	mutated.Challenge.CacheNodes++
	if got := service.Submit(mutated); got.Status != MiningSubmitMutated {
		t.Fatalf("mutated challenge status: got %q want %q", got.Status, MiningSubmitMutated)
	}
	if verified != 0 {
		t.Fatal("mutated staging work must be rejected before verifier execution")
	}

	invalid := MiningStagingSolution{Challenge: challenge, Nonce: 8}
	if got := service.Submit(invalid); got.Status != MiningSubmitInvalid {
		t.Fatalf("invalid nonce status: got %q want %q", got.Status, MiningSubmitInvalid)
	}
	if verified != 1 {
		t.Fatalf("verifier calls after invalid nonce: got %d want 1", verified)
	}

	valid := MiningStagingSolution{Challenge: challenge, Nonce: 7}
	if got := service.Submit(valid); got.Status != MiningSubmitAccepted {
		t.Fatalf("valid staging nonce status: got %q want %q", got.Status, MiningSubmitAccepted)
	}
	if verified != 2 {
		t.Fatalf("verifier calls after accepted nonce: got %d want 2", verified)
	}
}

func TestMiningStagingChallengeDoesNotSelectProductionCachePolicy(t *testing.T) {
	service := NewMiningStagingService(func(MiningStagingChallenge, uint64) bool { return false })
	if _, err := service.Issue([]byte("hardware-gate"), 0, 0, bytes.Repeat([]byte{0xff}, 32)); err == nil {
		t.Fatal("staging challenge must require an explicit non-zero cache node count")
	}
}
