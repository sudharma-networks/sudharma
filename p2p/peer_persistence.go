package p2p

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// KnownPeer represents a peer that this node has successfully
// communicated with and may reconnect to after restart.
type KnownPeer struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

// knownPeerFile is the on-disk representation of the peer database.
type knownPeerFile struct {
	Peers []KnownPeer `json:"peers"`
}

// SaveKnownPeersToFile persists known peers to disk.
//
// The supplied peer list is normalized before saving:
//   - empty Node IDs are rejected
//   - empty addresses are rejected
//   - the local node is rejected
//   - duplicate Node IDs are removed
//   - duplicate addresses are removed
func SaveKnownPeersToFile(
	path string,
	localNodeID string,
	peers []KnownPeer,
) error {

	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("peer file path cannot be empty")
	}

	if strings.TrimSpace(localNodeID) == "" {
		return fmt.Errorf("local node ID cannot be empty")
	}

	normalized := make(
		[]KnownPeer,
		0,
		len(peers),
	)

	seenNodeIDs := make(map[string]struct{})
	seenAddresses := make(map[string]struct{})

	for _, peer := range peers {

		nodeID := strings.TrimSpace(peer.NodeID)
		address := strings.TrimSpace(peer.Address)

		if nodeID == "" {
			return fmt.Errorf(
				"known peer node ID cannot be empty",
			)
		}

		if address == "" {
			return fmt.Errorf(
				"known peer address cannot be empty",
			)
		}

		if nodeID == localNodeID {
			return fmt.Errorf(
				"cannot persist local node as known peer",
			)
		}

		if _, exists := seenNodeIDs[nodeID]; exists {
			continue
		}

		if _, exists := seenAddresses[address]; exists {
			continue
		}

		seenNodeIDs[nodeID] = struct{}{}
		seenAddresses[address] = struct{}{}

		normalized = append(
			normalized,
			KnownPeer{
				NodeID:  nodeID,
				Address: address,
			},
		)
	}

	sort.Slice(
		normalized,
		func(i, j int) bool {
			return normalized[i].NodeID <
				normalized[j].NodeID
		},
	)

	data, err := json.MarshalIndent(
		knownPeerFile{
			Peers: normalized,
		},
		"",
		"  ",
	)

	if err != nil {
		return fmt.Errorf(
			"failed encoding known peers: %w",
			err,
		)
	}

	directory := filepath.Dir(path)

	if directory != "." {
		if err := os.MkdirAll(
			directory,
			0o755,
		); err != nil {

			return fmt.Errorf(
				"failed creating peer directory: %w",
				err,
			)
		}
	}

	temporaryPath := path + ".tmp"

	if err := os.WriteFile(
		temporaryPath,
		data,
		0o600,
	); err != nil {

		return fmt.Errorf(
			"failed writing temporary peer file: %w",
			err,
		)
	}

	if err := os.Rename(
		temporaryPath,
		path,
	); err != nil {

		_ = os.Remove(temporaryPath)

		return fmt.Errorf(
			"failed committing peer file: %w",
			err,
		)
	}

	return nil
}

// LoadKnownPeersFromFile loads and validates the persisted
// peer database.
//
// A missing file is treated as an empty peer database.
func LoadKnownPeersFromFile(
	path string,
	localNodeID string,
) ([]KnownPeer, error) {

	if strings.TrimSpace(path) == "" {
		return nil,
			fmt.Errorf(
				"peer file path cannot be empty",
			)
	}

	if strings.TrimSpace(localNodeID) == "" {
		return nil,
			fmt.Errorf(
				"local node ID cannot be empty",
			)
	}

	data, err := os.ReadFile(path)

	if err != nil {

		if os.IsNotExist(err) {
			return []KnownPeer{}, nil
		}

		return nil,
			fmt.Errorf(
				"failed reading known peer file: %w",
				err,
			)
	}

	var file knownPeerFile

	if err := json.Unmarshal(
		data,
		&file,
	); err != nil {

		return nil,
			fmt.Errorf(
				"failed decoding known peer file: %w",
				err,
			)
	}

	normalized := make(
		[]KnownPeer,
		0,
		len(file.Peers),
	)

	seenNodeIDs := make(map[string]struct{})
	seenAddresses := make(map[string]struct{})

	for _, peer := range file.Peers {

		nodeID := strings.TrimSpace(peer.NodeID)
		address := strings.TrimSpace(peer.Address)

		if nodeID == "" {
			return nil,
				fmt.Errorf(
					"persisted peer has empty node ID",
				)
		}

		if address == "" {
			return nil,
				fmt.Errorf(
					"persisted peer %s has empty address",
					nodeID,
				)
		}

		if nodeID == localNodeID {
			continue
		}

		if _, exists := seenNodeIDs[nodeID]; exists {
			continue
		}

		if _, exists := seenAddresses[address]; exists {
			continue
		}

		seenNodeIDs[nodeID] = struct{}{}
		seenAddresses[address] = struct{}{}

		normalized = append(
			normalized,
			KnownPeer{
				NodeID:  nodeID,
				Address: address,
			},
		)
	}

	sort.Slice(
		normalized,
		func(i, j int) bool {
			return normalized[i].NodeID <
				normalized[j].NodeID
		},
	)

	return normalized, nil
}
