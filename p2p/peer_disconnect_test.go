package p2p

import (
	"testing"
	"time"
)

func TestDeadPeerRemovedAfterRemoteShutdown(t *testing.T) {
	nodeA, err := NewNode(
		"disconnect-a",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"disconnect-b",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeA.Stop()

	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeA.Connect(
		nodeB.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(
		2 * time.Second,
	)

	for time.Now().Before(deadline) {
		if nodeA.PeerCount() == 1 &&
			nodeB.PeerCount() == 1 {

			break
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if nodeA.PeerCount() != 1 {
		t.Fatalf(
			"expected node A to have 1 peer before shutdown, got %d",
			nodeA.PeerCount(),
		)
	}

	if nodeB.PeerCount() != 1 {
		t.Fatalf(
			"expected node B to have 1 peer before shutdown, got %d",
			nodeB.PeerCount(),
		)
	}

	if err := nodeB.Stop(); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(
		2 * time.Second,
	)

	for time.Now().Before(deadline) {
		if nodeA.PeerCount() == 0 {
			return
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	t.Fatalf(
		"dead peer was not removed; node A still has %d peer(s)",
		nodeA.PeerCount(),
	)
}
