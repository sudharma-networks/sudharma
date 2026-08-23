package p2p

import "sync"

const (
	PeerScoreInitial             = 0
	PeerScoreMin                 = -100
	PeerScoreMax                 = 100
	PeerScoreGoodEvent           = 1
	PeerPenaltyConnectionFailure = 2
	PeerPenaltyMalformed         = 10
	PeerPenaltyInvalidData       = 20
	PeerPenaltyProtocolAbuse     = 30
	PeerDisconnectThreshold      = -50
	PeerAvoidThreshold           = -30
)

// PeerScorer tracks bounded reputation scores for remote peers.
// Scores are local-only policy and are never part of consensus.
type PeerScorer struct {
	mu     sync.RWMutex
	scores map[string]int
}

func NewPeerScorer() *PeerScorer {
	return &PeerScorer{scores: make(map[string]int)}
}

func (s *PeerScorer) Score(nodeID string) int {
	if s == nil || nodeID == "" {
		return PeerScoreInitial
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scores[nodeID]
}

func (s *PeerScorer) Reward(nodeID string, amount int) int {
	if s == nil || nodeID == "" || amount <= 0 {
		return s.Score(nodeID)
	}
	return s.adjust(nodeID, amount)
}

func (s *PeerScorer) Penalize(nodeID string, amount int) int {
	if s == nil || nodeID == "" || amount <= 0 {
		return s.Score(nodeID)
	}
	return s.adjust(nodeID, -amount)
}

func (s *PeerScorer) ShouldAvoid(nodeID string) bool {
	return s.Score(nodeID) <= PeerAvoidThreshold
}

func (s *PeerScorer) ShouldDisconnect(nodeID string) bool {
	return s.Score(nodeID) <= PeerDisconnectThreshold
}

func (s *PeerScorer) Remove(nodeID string) {
	if s == nil || nodeID == "" {
		return
	}
	s.mu.Lock()
	delete(s.scores, nodeID)
	s.mu.Unlock()
}

func (s *PeerScorer) adjust(nodeID string, delta int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.scores[nodeID] + delta
	if next > PeerScoreMax {
		next = PeerScoreMax
	}
	if next < PeerScoreMin {
		next = PeerScoreMin
	}
	s.scores[nodeID] = next
	return next
}
