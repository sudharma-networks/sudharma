package p2p

import (
	"testing"
	"time"
)

func TestRecoverNetworkOnceRejectsInvalidTimeout(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearPartitionRecoveryState(node)

	if _, err := node.RecoverNetworkOnce(0); err == nil {
		t.Fatal("expected invalid recovery timeout to fail")
	}
}

func TestRecoverNetworkOnceReportsNoConnectedPeers(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer clearPartitionRecoveryState(node)

	result, err := node.RecoverNetworkOnce(time.Millisecond)
	if err == nil {
		t.Fatal("expected recovery without peers to fail")
	}
	if result == nil {
		t.Fatal("expected recovery result even when no peers are available")
	}
	if len(result.AttemptedPeers) != 0 {
		t.Fatalf("expected no attempted peers, got %v", result.AttemptedPeers)
	}
}
