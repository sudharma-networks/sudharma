package p2p

import (
	"fmt"
	"sort"
)

// PeerDiscoverySnapshot returns a safe snapshot of currently connected
// peers that may be shared with another Sudharma Network peer.
//
// The local node and the requesting peer are excluded.
func (n *Node) PeerDiscoverySnapshot(
	excludeNodeID string,
) []KnownPeer {

	n.mu.RLock()
	defer n.mu.RUnlock()

	result := make(
		[]KnownPeer,
		0,
		len(n.peers),
	)

	for nodeID, peer := range n.peers {

		if peer == nil {
			continue
		}

		if nodeID == "" ||
			nodeID == n.NodeID ||
			nodeID == excludeNodeID {

			continue
		}

		address := peer.Info.ListenAddress

		if address == "" {
			continue
		}

		result = append(
			result,
			KnownPeer{
				NodeID:  nodeID,
				Address: address,
			},
		)
	}

	sort.Slice(
		result,
		func(i, j int) bool {
			return result[i].NodeID <
				result[j].NodeID
		},
	)

	if len(result) > MaxPeersPerMessage {
		result = result[:MaxPeersPerMessage]
	}

	return result
}

// SendGetPeers asks one connected peer for its peer-discovery snapshot.
func (n *Node) SendGetPeers(
	nodeID string,
) error {

	if nodeID == "" {
		return fmt.Errorf(
			"peer node ID cannot be empty",
		)
	}

	n.mu.RLock()
	peer, ok := n.peers[nodeID]
	n.mu.RUnlock()

	if !ok || peer == nil {
		return fmt.Errorf(
			"peer not found: %s",
			nodeID,
		)
	}

	data, err := NewGetPeersMessage()
	if err != nil {
		return err
	}

	if err := peer.write(data); err != nil {
		return fmt.Errorf(
			"failed requesting peers from %s: %w",
			nodeID,
			err,
		)
	}

	return nil
}

// sendPeersToPeer responds to one peer-discovery request.
func (n *Node) sendPeersToPeer(
	peer *PeerConnection,
) (int, error) {

	if peer == nil {
		return 0, fmt.Errorf(
			"peer cannot be nil",
		)
	}

	peers := n.PeerDiscoverySnapshot(
		peer.Info.NodeID,
	)

	data, err := NewPeersMessage(peers)
	if err != nil {
		return 0, err
	}

	if err := peer.write(data); err != nil {
		return 0,
			fmt.Errorf(
				"failed sending peers to %s: %w",
				peer.Info.NodeID,
				err,
			)
	}

	return len(peers), nil
}

// filterDiscoveredPeers removes entries that are unsafe or useless
// for the local node. The returned list is deterministic.
func (n *Node) filterDiscoveredPeers(
	peers []KnownPeer,
) []KnownPeer {

	n.mu.RLock()

	localNodeID := n.NodeID
	localAddress := n.ListenAddress

	connectedNodeIDs := make(
		map[string]struct{},
		len(n.peers),
	)

	connectedAddresses := make(
		map[string]struct{},
		len(n.peers),
	)

	for nodeID, peer := range n.peers {
		connectedNodeIDs[nodeID] = struct{}{}

		if peer != nil &&
			peer.Info.ListenAddress != "" {

			connectedAddresses[peer.Info.ListenAddress] = struct{}{}
		}
	}

	n.mu.RUnlock()

	result := make(
		[]KnownPeer,
		0,
		len(peers),
	)

	seenNodeIDs := make(map[string]struct{})
	seenAddresses := make(map[string]struct{})

	for _, peer := range peers {

		if peer.NodeID == "" ||
			peer.Address == "" {

			continue
		}

		if peer.NodeID == localNodeID ||
			peer.Address == localAddress {

			continue
		}

		if _, exists := connectedNodeIDs[peer.NodeID]; exists {
			continue
		}

		if _, exists := connectedAddresses[peer.Address]; exists {
			continue
		}

		if _, exists := seenNodeIDs[peer.NodeID]; exists {
			continue
		}

		if _, exists := seenAddresses[peer.Address]; exists {
			continue
		}

		seenNodeIDs[peer.NodeID] = struct{}{}
		seenAddresses[peer.Address] = struct{}{}

		result = append(
			result,
			peer,
		)
	}

	sort.Slice(
		result,
		func(i, j int) bool {
			return result[i].NodeID <
				result[j].NodeID
		},
	)

	return result
}

func (n *Node) connectedPeerAddresses() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	addresses := make([]string, 0, len(n.peers))
	for _, peer := range n.peers {
		if peer == nil || peer.Info.ListenAddress == "" {
			continue
		}
		addresses = append(addresses, peer.Info.ListenAddress)
	}
	return addresses
}

// AutoConnectDiscoveredPeers attempts connections to discovered peers while
// limiting concentration from any single network group.
//
// Failed peers remain in the discovered set so a later discovery or reconnect
// cycle can try them again.
func (n *Node) AutoConnectDiscoveredPeers() (
	connected int,
	failed int,
) {

	discovered := n.DiscoveredPeersSnapshot()
	selectedAddresses := n.connectedPeerAddresses()

	for _, candidate := range discovered {

		if candidate.NodeID == "" ||
			candidate.Address == "" {

			continue
		}

		n.mu.RLock()

		_, alreadyConnected :=
			n.peers[candidate.NodeID]

		localNodeID := n.NodeID
		localAddress := n.ListenAddress

		n.mu.RUnlock()

		if alreadyConnected ||
			candidate.NodeID == localNodeID ||
			candidate.Address == localAddress {

			continue
		}

		if !CanAddPeerFromNetworkGroup(selectedAddresses, candidate.Address) {
			fmt.Printf(
				"[PEERS] Skipping discovered peer %s at %s: network diversity limit reached\n",
				candidate.NodeID,
				candidate.Address,
			)
			continue
		}

		fmt.Printf(
			"[PEERS] Auto-connecting to discovered peer %s at %s...\n",
			candidate.NodeID,
			candidate.Address,
		)

		peer, err :=
			n.Connect(
				candidate.Address,
			)

		if err != nil {
			failed++

			fmt.Printf(
				"[PEERS] Auto-connect failed for %s: %v\n",
				candidate.NodeID,
				err,
			)

			continue
		}

		connected++
		selectedAddresses = append(selectedAddresses, candidate.Address)

		fmt.Printf(
			"[PEERS] Auto-connected to %s at %s\n",
			peer.NodeID,
			candidate.Address,
		)
	}

	return connected, failed
}
