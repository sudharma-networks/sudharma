package p2p

import "testing"

func TestPeerScorerStartsNeutral(t *testing.T) {
	s := NewPeerScorer()
	if got := s.Score("peer-a"); got != 0 {
		t.Fatalf("expected neutral score 0, got %d", got)
	}
}

func TestPeerScorerRewardsAndPenalizes(t *testing.T) {
	s := NewPeerScorer()
	if got := s.Reward("peer-a", 5); got != 5 {
		t.Fatalf("expected score 5, got %d", got)
	}
	if got := s.Penalize("peer-a", 3); got != 2 {
		t.Fatalf("expected score 2, got %d", got)
	}
}

func TestPeerScorerIsBounded(t *testing.T) {
	s := NewPeerScorer()
	if got := s.Reward("peer-a", 1000); got != PeerScoreMax {
		t.Fatalf("expected max score %d, got %d", PeerScoreMax, got)
	}
	if got := s.Penalize("peer-a", 1000); got != PeerScoreMin {
		t.Fatalf("expected min score %d, got %d", PeerScoreMin, got)
	}
}

func TestPeerScorerAvoidAndDisconnectThresholds(t *testing.T) {
	s := NewPeerScorer()
	s.Penalize("peer-a", 30)
	if !s.ShouldAvoid("peer-a") {
		t.Fatal("expected peer to be avoided at threshold")
	}
	if s.ShouldDisconnect("peer-a") {
		t.Fatal("peer should not disconnect yet")
	}
	s.Penalize("peer-a", 20)
	if !s.ShouldDisconnect("peer-a") {
		t.Fatal("expected peer to disconnect at threshold")
	}
}

func TestPeerScorerRemove(t *testing.T) {
	s := NewPeerScorer()
	s.Penalize("peer-a", 40)
	s.Remove("peer-a")
	if got := s.Score("peer-a"); got != 0 {
		t.Fatalf("expected removed score to return neutral, got %d", got)
	}
}
