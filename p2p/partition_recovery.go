package p2p

import "sort"

// DiversifiedPeerOrder returns a deterministic round-robin ordering of known
// peers across network groups. This reduces the chance that reconnect logic
// repeatedly consumes all early connection attempts inside one isolated
// address range after a restart or partial network partition.
func DiversifiedPeerOrder(peers []KnownPeer) []KnownPeer {
	groups := make(map[string][]KnownPeer)
	groupNames := make([]string, 0)
	seenNodeIDs := make(map[string]struct{})
	seenAddresses := make(map[string]struct{})

	for _, peer := range peers {
		if peer.NodeID == "" || peer.Address == "" {
			continue
		}
		if _, exists := seenNodeIDs[peer.NodeID]; exists {
			continue
		}
		if _, exists := seenAddresses[peer.Address]; exists {
			continue
		}

		group, err := PeerNetworkGroup(peer.Address)
		if err != nil {
			continue
		}

		seenNodeIDs[peer.NodeID] = struct{}{}
		seenAddresses[peer.Address] = struct{}{}
		if _, exists := groups[group]; !exists {
			groupNames = append(groupNames, group)
		}
		groups[group] = append(groups[group], peer)
	}

	sort.Strings(groupNames)
	for _, group := range groupNames {
		sort.Slice(groups[group], func(i, j int) bool {
			if groups[group][i].NodeID == groups[group][j].NodeID {
				return groups[group][i].Address < groups[group][j].Address
			}
			return groups[group][i].NodeID < groups[group][j].NodeID
		})
	}

	ordered := make([]KnownPeer, 0, len(seenNodeIDs))
	for round := 0; ; round++ {
		added := false
		for _, group := range groupNames {
			bucket := groups[group]
			if round >= len(bucket) {
				continue
			}
			ordered = append(ordered, bucket[round])
			added = true
		}
		if !added {
			break
		}
	}

	return ordered
}
