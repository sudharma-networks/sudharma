package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func offlineMainnetChainForTest(t *testing.T) *Chain {
	t.Helper()

	chain, err := newChainFromGenesisForNetwork(
		params.NetworkMainnet,
		NewMainnetGenesisBlock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func TestCloneChainPreservesNetworkIdentity(t *testing.T) {
	source := offlineMainnetChainForTest(t)

	clone, err := CloneChain(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := clone.Network(); got != params.NetworkMainnet {
		t.Fatalf("clone network = %q, want %q", got, params.NetworkMainnet)
	}
}

func TestForkChoiceRejectsCrossNetworkCandidate(t *testing.T) {
	current := NewChain()
	candidate := offlineMainnetChainForTest(t)

	if _, err := BetterChain(current, candidate); err == nil {
		t.Fatal("expected cross-network fork choice to fail closed")
	}
}

func TestReplaceWithRejectsCrossNetworkCandidate(t *testing.T) {
	current := NewChain()
	candidate := offlineMainnetChainForTest(t)

	if err := current.ReplaceWith(candidate); err == nil {
		t.Fatal("expected cross-network replacement to fail closed")
	}
	if got := current.Network(); got != params.NetworkPublicTestnet {
		t.Fatalf("current network changed to %q", got)
	}
}

func TestValidateAndCloneChainPreservesOfflineMainnetIdentity(t *testing.T) {
	source := offlineMainnetChainForTest(t)

	validated, err := ValidateAndCloneChain(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := validated.Network(); got != params.NetworkMainnet {
		t.Fatalf("validated network = %q, want %q", got, params.NetworkMainnet)
	}
}

func TestBuildStateFromChainUsesChainMonetaryPolicy(t *testing.T) {
	chain := offlineMainnetChainForTest(t)

	state, err := BuildStateFromChain(chain)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.MonetaryPolicy(); got != params.MonetaryPolicyMainnet {
		t.Fatalf("state policy = %d, want %d", got, params.MonetaryPolicyMainnet)
	}
}
