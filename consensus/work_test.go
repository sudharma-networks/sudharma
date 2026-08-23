package consensus

import "testing"

func TestWorkFromDifficultyZeroIsZero(t *testing.T) {
	work := WorkFromDifficulty(0)
	if work.Sign() != 0 {
		t.Fatalf("expected zero work for difficulty 0, got %s", work.String())
	}
}

func TestHigherDifficultyProducesMoreWork(t *testing.T) {
	low := WorkFromDifficulty(1)
	high := WorkFromDifficulty(10)
	if high.Cmp(low) <= 0 {
		t.Fatalf("higher difficulty should produce more work: low=%s high=%s", low.String(), high.String())
	}
}

func TestWorkFromDifficultyDeterministic(t *testing.T) {
	first := WorkFromDifficulty(100)
	second := WorkFromDifficulty(100)
	if first.Cmp(second) != 0 {
		t.Fatalf("work calculation is not deterministic: %s != %s", first.String(), second.String())
	}
}

func TestTargetFromDifficultyZeroIsZero(t *testing.T) {
	target := TargetFromDifficulty(0)
	if target.Sign() != 0 {
		t.Fatalf("expected zero target for difficulty 0, got %s", target.String())
	}
}
