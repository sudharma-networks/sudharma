package rpc

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

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
	if got := service.Submit(valid); got.Status != MiningSubmitStale {
		t.Fatalf("replayed staging nonce status: got %q want %q", got.Status, MiningSubmitStale)
	}
}

func TestMiningStagingKeepsIndependentOutstandingChallenges(t *testing.T) {
	service := NewMiningStagingService(func(challenge MiningStagingChallenge, nonce uint64) bool {
		return nonce == 7
	})
	target := bytes.Repeat([]byte{0xff}, 32)

	first, err := service.Issue([]byte("hardware-gate-first"), 0, 8, target)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Issue([]byte("hardware-gate-second"), 0, 8, target)
	if err != nil {
		t.Fatal(err)
	}

	if got := service.Submit(MiningStagingSolution{Challenge: first, Nonce: 7}); got.Status != MiningSubmitAccepted {
		t.Fatalf("first outstanding challenge status: got %q want %q", got.Status, MiningSubmitAccepted)
	}
	if got := service.Submit(MiningStagingSolution{Challenge: second, Nonce: 7}); got.Status != MiningSubmitAccepted {
		t.Fatalf("second outstanding challenge status: got %q want %q", got.Status, MiningSubmitAccepted)
	}
}

func TestMiningStagingBoundsOutstandingChallenges(t *testing.T) {
	service := NewMiningStagingService(func(challenge MiningStagingChallenge, nonce uint64) bool {
		return nonce == 7
	})
	target := bytes.Repeat([]byte{0xff}, 32)
	issued := make([]MiningStagingChallenge, 0, miningStagingMaxOutstanding+1)
	for i := 0; i < miningStagingMaxOutstanding+1; i++ {
		challenge, err := service.Issue([]byte{byte(i + 1)}, 0, 8, target)
		if err != nil {
			t.Fatal(err)
		}
		issued = append(issued, challenge)
	}

	if got := service.Submit(MiningStagingSolution{Challenge: issued[0], Nonce: 7}); got.Status != MiningSubmitStale {
		t.Fatalf("evicted staging challenge status: got %q want %q", got.Status, MiningSubmitStale)
	}
	if got := service.Submit(MiningStagingSolution{Challenge: issued[len(issued)-1], Nonce: 7}); got.Status != MiningSubmitAccepted {
		t.Fatalf("newest staging challenge status: got %q want %q", got.Status, MiningSubmitAccepted)
	}
}

func TestMiningStagingExpiresOutstandingChallenges(t *testing.T) {
	originalNow := miningStagingNow
	defer func() { miningStagingNow = originalNow }()

	now := time.Unix(1_800_000_000, 0)
	miningStagingNow = func() time.Time { return now }
	service := NewMiningStagingService(func(challenge MiningStagingChallenge, nonce uint64) bool {
		return nonce == 7
	})
	challenge, err := service.Issue([]byte("hardware-gate-expiry"), 0, 8, bytes.Repeat([]byte{0xff}, 32))
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(miningStagingChallengeTTL + time.Nanosecond)
	if got := service.Submit(MiningStagingSolution{Challenge: challenge, Nonce: 7}); got.Status != MiningSubmitStale {
		t.Fatalf("expired staging challenge status: got %q want %q", got.Status, MiningSubmitStale)
	}
}

func TestMiningStagingChallengeDoesNotSelectProductionCachePolicy(t *testing.T) {
	service := NewMiningStagingService(func(MiningStagingChallenge, uint64) bool { return false })
	if _, err := service.Issue([]byte("hardware-gate"), 0, 0, bytes.Repeat([]byte{0xff}, 32)); err == nil {
		t.Fatal("staging challenge must require an explicit non-zero cache node count")
	}
}
