package p2p

import (
	"testing"
	"time"
)

func TestPeerDiscoveryRequestResponse(t *testing.T) {

	nodeA, err := NewNode(
		"discovery-a",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeB, err := NewNode(
		"discovery-b",
		"127.0.0.1:0",
		0,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	nodeC, err := NewNode(
		"discovery-c",
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
	defer nodeB.Stop()

	if err := nodeC.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeC.Stop()

	if _, err := nodeB.Connect(
		nodeC.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeA.Connect(
		nodeB.ListenAddress,
	); err != nil {
		t.Fatal(err)
	}

	if err :=
		nodeA.SendGetPeers(
			"discovery-b",
		); err != nil {

		t.Fatal(err)
	}

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for time.Now().Before(
		deadline,
	) {

		discovered :=
			nodeA.DiscoveredPeersSnapshot()

		if len(discovered) == 1 {
			if discovered[0].NodeID !=
				"discovery-c" {

				t.Fatalf(
					"expected discovery-c, got %s",
					discovered[0].NodeID,
				)
			}

			if discovered[0].Address !=
				nodeC.ListenAddress {

				t.Fatalf(
					"expected %s, got %s",
					nodeC.ListenAddress,
					discovered[0].Address,
				)
			}

			return
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	t.Fatal(
		"node A did not learn node C through node B",
	)
}

func TestPeerDiscoveryFiltersSelfAndConnectedPeers(t *testing.T) {

	node, err :=
		NewNode(
			"discovery-local",
			"127.0.0.1:19000",
			0,
			"",
		)

	if err != nil {
		t.Fatal(err)
	}

	input := []KnownPeer{
		{
			NodeID:  "discovery-local",
			Address: "127.0.0.1:19000",
		},
		{
			NodeID:  "peer-c",
			Address: "127.0.0.1:19003",
		},
		{
			NodeID:  "peer-c",
			Address: "127.0.0.1:19099",
		},
	}

	filtered :=
		node.filterDiscoveredPeers(
			input,
		)

	if len(filtered) != 1 {
		t.Fatalf(
			"expected 1 safe discovered peer, got %d",
			len(filtered),
		)
	}

	if filtered[0].NodeID != "peer-c" {
		t.Fatalf(
			"expected peer-c, got %s",
			filtered[0].NodeID,
		)
	}
}
