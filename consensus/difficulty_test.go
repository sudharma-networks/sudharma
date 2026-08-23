package consensus

import "testing"

func TestDifficultyIncreasesForFastBlocks(t *testing.T) {
	current := uint32(100)

	next := NextDifficulty(
		current,
		20,
	)

	if next <= current {
		t.Fatalf(
			"expected difficulty to increase: current %d, next %d",
			current,
			next,
		)
	}
}

func TestDifficultyDecreasesForSlowBlocks(t *testing.T) {
	current := uint32(100)

	next := NextDifficulty(
		current,
		180,
	)

	if next >= current {
		t.Fatalf(
			"expected difficulty to decrease: current %d, next %d",
			current,
			next,
		)
	}
}

func TestDifficultyStableNearTarget(t *testing.T) {
	current := uint32(100)

	next := NextDifficulty(
		current,
		60,
	)

	if next != current {
		t.Fatalf(
			"expected difficulty %d, got %d",
			current,
			next,
		)
	}
}

func TestDifficultyNeverBelowOne(t *testing.T) {
	next := NextDifficulty(
		1,
		1000,
	)

	if next != 1 {
		t.Fatalf(
			"expected minimum difficulty 1, got %d",
			next,
		)
	}
}

func TestZeroDifficultyBecomesOne(t *testing.T) {
	next := NextDifficulty(
		0,
		60,
	)

	if next != 1 {
		t.Fatalf(
			"expected difficulty 1, got %d",
			next,
		)
	}
}
