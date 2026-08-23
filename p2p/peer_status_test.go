package p2p

import (
	"testing"
	"time"
)

func TestIsPeerConnectedTracksDisconnect(t *testing.T) {
	nodeA, err := NewNode(
		"status-a",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"status-b",
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
		if nodeA.IsPeerConnected(
			"status-b",
		) {
			break
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if !nodeA.IsPeerConnected(
		"status-b",
	) {
		t.Fatal(
			"expected status-b to be connected",
		)
	}

	if err := nodeB.Stop(); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(
		2 * time.Second,
	)

	for time.Now().Before(deadline) {
		if !nodeA.IsPeerConnected(
			"status-b",
		) {
			return
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	t.Fatal(
		"peer remained marked connected after remote shutdown",
	)
}
