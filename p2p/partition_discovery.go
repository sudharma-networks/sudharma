package p2p

import "sort"

const MaxPartitionRecoveryDiscoverySources = 8

// partitionDiscoveryCandidates returns at most one connected peer from each
// network group. Asking independent groups for peer lists makes discovery less
// dependent on a single potentially partitioned or adversarial source.
func (n *Node) partitionDiscoveryCandidates() []*PeerConnection {
	if n == nil {
		return nil
	}

	n.mu.RLock()
	candidates := make([]*PeerConnection, 0, len(n.peers))
	for _, peer := range n.peers {
		if peer == nil || peer.Info.NodeID == "" || n.shouldAvoidPeer(peer.Info.NodeID) {
			continue
		}
		candidates = append(candidates, peer)
	}
	n.mu.RUnlock()

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Info.NodeID < candidates[j].Info.NodeID
	})

	selected := make([]*PeerConnection, 0, len(candidates))
	seenGroups := make(map[string]struct{})
	for _, peer := range candidates {
		group, err := PeerNetworkGroup(peerNetworkAddress(peer))
		if err != nil {
			continue
		}
		if _, exists := seenGroups[group]; exists {
			continue
		}
		seenGroups[group] = struct{}{}
		selected = append(selected, peer)
		if len(selected) >= MaxPartitionRecoveryDiscoverySources {
			break
		}
	}

	return selected
}

// RequestPartitionRecoveryPeers asks diverse connected peers for independent
// discovery snapshots. Failure of one source does not stop requests to others.
func (n *Node) RequestPartitionRecoveryPeers() (requested int, failed int) {
	for _, peer := range n.partitionDiscoveryCandidates() {
		data, err := NewGetPeersMessage()
		if err != nil {
			failed++
			continue
		}
		if err := peer.write(data); err != nil {
			failed++
			continue
		}
		requested++
	}
	return requested, failed
}
