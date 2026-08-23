package p2p

import (
	"fmt"
	"math/big"
	"sort"
	"time"
)

const MaxPartitionRecoveryPeers = 8

// RecoveryCandidate is a local snapshot used to choose which connected peers
// should be tried during partition recovery. Advertised work is only a ranking
// hint: SyncFromPeer still downloads and validates the candidate chain locally
// before fork choice or reorganization.
type RecoveryCandidate struct {
	NodeID     string
	Height     uint64
	TotalWork  *big.Int
	PeerScore  int
}

// PartitionRecoveryResult describes one recovery cycle.
type PartitionRecoveryResult struct {
	AttemptedPeers []string
	SucceededPeer  string
	Failures       map[string]string
}

// recoveryCandidates returns connected, non-avoided peers ordered by advertised
// cumulative work, then height, local reputation and Node ID. Invalid advertised
// work is excluded and penalized as invalid peer data.
func (n *Node) recoveryCandidates() []RecoveryCandidate {
	if n == nil {
		return nil
	}

	type peerSnapshot struct {
		nodeID    string
		height    uint64
		totalWork string
	}

	n.mu.RLock()
	snapshots := make([]peerSnapshot, 0, len(n.peers))
	for nodeID, peer := range n.peers {
		if peer == nil || nodeID == "" {
			continue
		}
		snapshots = append(snapshots, peerSnapshot{
			nodeID:    nodeID,
			height:    peer.Info.Height,
			totalWork: peer.Info.TotalWork,
		})
	}
	n.mu.RUnlock()

	candidates := make([]RecoveryCandidate, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if n.shouldAvoidPeer(snapshot.nodeID) {
			continue
		}

		work := new(big.Int)
		if _, ok := work.SetString(snapshot.totalWork, 10); !ok || work.Sign() < 0 {
			n.penalizePeer(snapshot.nodeID, PeerPenaltyInvalidData)
			continue
		}

		candidates = append(candidates, RecoveryCandidate{
			NodeID:    snapshot.nodeID,
			Height:    snapshot.height,
			TotalWork: new(big.Int).Set(work),
			PeerScore: n.PeerScore(snapshot.nodeID),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if cmp := candidates[i].TotalWork.Cmp(candidates[j].TotalWork); cmp != 0 {
			return cmp > 0
		}
		if candidates[i].Height != candidates[j].Height {
			return candidates[i].Height > candidates[j].Height
		}
		if candidates[i].PeerScore != candidates[j].PeerScore {
			return candidates[i].PeerScore > candidates[j].PeerScore
		}
		return candidates[i].NodeID < candidates[j].NodeID
	})

	if len(candidates) > MaxPartitionRecoveryPeers {
		candidates = candidates[:MaxPartitionRecoveryPeers]
	}
	return candidates
}

// SyncFromBestAvailablePeer performs one bounded partition-recovery cycle.
// It never trusts advertised work for adoption: each attempted peer is passed
// through the existing SyncFromPeer validation and fork-choice path. If one
// peer fails, the node tries the next independent connected candidate instead
// of allowing a single dishonest or unreachable peer to block convergence.
func (n *Node) SyncFromBestAvailablePeer(timeout time.Duration) (*PartitionRecoveryResult, error) {
	if n == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("sync timeout must be greater than zero")
	}

	result := &PartitionRecoveryResult{Failures: make(map[string]string)}
	candidates := n.recoveryCandidates()
	if len(candidates) == 0 {
		return result, fmt.Errorf("no eligible connected peers available for partition recovery")
	}

	for _, candidate := range candidates {
		result.AttemptedPeers = append(result.AttemptedPeers, candidate.NodeID)

		if err := n.SyncFromPeer(candidate.NodeID, timeout); err != nil {
			result.Failures[candidate.NodeID] = err.Error()
			// A failed recovery attempt can be caused by disconnects as well as
			// bad remote data, so apply only the small connection-failure penalty
			// here. Deeper protocol validators apply stronger penalties themselves.
			n.penalizePeer(candidate.NodeID, PeerPenaltyConnectionFailure)
			continue
		}

		n.rewardPeer(candidate.NodeID, PeerScoreGoodEvent)
		result.SucceededPeer = candidate.NodeID
		return result, nil
	}

	return result, fmt.Errorf("partition recovery failed with all %d eligible peer(s)", len(candidates))
}
