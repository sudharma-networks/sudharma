package p2p

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKnownPeersSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"sudharma-peers.json",
	)

	peers := []KnownPeer{
		{
			NodeID:  "peer-b",
			Address: "127.0.0.1:19002",
		},
		{
			NodeID:  "peer-a",
			Address: "127.0.0.1:19001",
		},
	}

	if err := SaveKnownPeersToFile(
		path,
		"local-node",
		peers,
	); err != nil {

		t.Fatal(err)
	}

	loaded, err :=
		LoadKnownPeersFromFile(
			path,
			"local-node",
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 2 {
		t.Fatalf(
			"expected 2 peers, got %d",
			len(loaded),
		)
	}

	// Save routine sorts by NodeID.
	if loaded[0].NodeID != "peer-a" {
		t.Fatalf(
			"expected peer-a first, got %s",
			loaded[0].NodeID,
		)
	}

	if loaded[1].NodeID != "peer-b" {
		t.Fatalf(
			"expected peer-b second, got %s",
			loaded[1].NodeID,
		)
	}
}

func TestKnownPeersMissingFileCreatesEmptyList(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"missing-peers.json",
	)

	loaded, err :=
		LoadKnownPeersFromFile(
			path,
			"local-node",
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 0 {
		t.Fatalf(
			"expected empty peer list, got %d",
			len(loaded),
		)
	}
}

func TestKnownPeersCorruptedFileRejected(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"sudharma-peers.json",
	)

	if err := os.WriteFile(
		path,
		[]byte("{not-valid-json"),
		0o600,
	); err != nil {

		t.Fatal(err)
	}

	_, err :=
		LoadKnownPeersFromFile(
			path,
			"local-node",
		)

	if err == nil {
		t.Fatal(
			"expected corrupted peer file to be rejected",
		)
	}
}

func TestKnownPeersSelfEntryIgnoredOnLoad(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"sudharma-peers.json",
	)

	data := []byte(`{
  "peers": [
    {
      "node_id": "local-node",
      "address": "127.0.0.1:19000"
    },
    {
      "node_id": "peer-b",
      "address": "127.0.0.1:19002"
    }
  ]
}`)

	if err := os.WriteFile(
		path,
		data,
		0o600,
	); err != nil {

		t.Fatal(err)
	}

	loaded, err :=
		LoadKnownPeersFromFile(
			path,
			"local-node",
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {
		t.Fatalf(
			"expected 1 non-self peer, got %d",
			len(loaded),
		)
	}

	if loaded[0].NodeID != "peer-b" {
		t.Fatalf(
			"expected peer-b, got %s",
			loaded[0].NodeID,
		)
	}
}

func TestKnownPeersDuplicateNodeIDRemoved(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"sudharma-peers.json",
	)

	peers := []KnownPeer{
		{
			NodeID:  "peer-a",
			Address: "127.0.0.1:19001",
		},
		{
			NodeID:  "peer-a",
			Address: "127.0.0.1:19099",
		},
	}

	if err := SaveKnownPeersToFile(
		path,
		"local-node",
		peers,
	); err != nil {

		t.Fatal(err)
	}

	loaded, err :=
		LoadKnownPeersFromFile(
			path,
			"local-node",
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {
		t.Fatalf(
			"expected duplicate NodeID to collapse to 1 peer, got %d",
			len(loaded),
		)
	}
}

func TestKnownPeersDuplicateAddressRemoved(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"sudharma-peers.json",
	)

	peers := []KnownPeer{
		{
			NodeID:  "peer-a",
			Address: "127.0.0.1:19001",
		},
		{
			NodeID:  "peer-b",
			Address: "127.0.0.1:19001",
		},
	}

	if err := SaveKnownPeersToFile(
		path,
		"local-node",
		peers,
	); err != nil {

		t.Fatal(err)
	}

	loaded, err :=
		LoadKnownPeersFromFile(
			path,
			"local-node",
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {
		t.Fatalf(
			"expected duplicate address to collapse to 1 peer, got %d",
			len(loaded),
		)
	}
}

func TestKnownPeersRejectsSelfOnSave(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"sudharma-peers.json",
	)

	peers := []KnownPeer{
		{
			NodeID:  "local-node",
			Address: "127.0.0.1:19000",
		},
	}

	err := SaveKnownPeersToFile(
		path,
		"local-node",
		peers,
	)

	if err == nil {
		t.Fatal(
			"expected local node to be rejected from persisted peers",
		)
	}
}
