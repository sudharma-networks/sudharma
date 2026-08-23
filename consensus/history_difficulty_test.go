package consensus

import "testing"

func TestMedianBlockIntervalDefaultsToTarget(t *testing.T) {
	got := MedianBlockInterval(nil)
	if got != 60 {
		t.Fatalf("expected default interval 60, got %d", got)
	}
}

func TestMedianBlockIntervalStable(t *testing.T) {
	got := MedianBlockInterval([]int64{55, 58, 60, 60, 61, 62, 63})
	if got != 60 {
		t.Fatalf("expected median 60, got %d", got)
	}
}

func TestMedianBlockIntervalResistsExtremeOutlier(t *testing.T) {
	got := MedianBlockInterval([]int64{58, 59, 60, 60, 61, 62, 999999})
	if got != 60 {
		t.Fatalf("extreme outlier changed median: expected 60, got %d", got)
	}
}

func TestHistoryDifficultyIncreasesForFastMedian(t *testing.T) {
	next := NextDifficultyFromHistory(100, []int64{10, 12, 15, 18, 20, 22, 25})
	if next <= 100 {
		t.Fatalf("expected difficulty increase, got %d", next)
	}
}

func TestHistoryDifficultyDecreasesForSlowMedian(t *testing.T) {
	next := NextDifficultyFromHistory(100, []int64{150, 160, 170, 180, 190, 200, 210})
	if next >= 100 {
		t.Fatalf("expected difficulty decrease, got %d", next)
	}
}

func TestHistoryDifficultyStableNearTarget(t *testing.T) {
	next := NextDifficultyFromHistory(100, []int64{55, 58, 60, 60, 61, 62, 65})
	if next != 100 {
		t.Fatalf("expected stable difficulty 100, got %d", next)
	}
}
