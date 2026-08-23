package consensus

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestBlockSubsidy(t *testing.T) {
	tests := []struct {
		height   uint64
		expected uint64
	}{
		{0, 50 * params.CoinDecimals},
		{999_999, 50 * params.CoinDecimals},
		{1_000_000, 25 * params.CoinDecimals},
		{2_000_000, 12_5000_0000},
		{3_000_000, 6_2500_0000},
	}

	for _, test := range tests {
		got := BlockSubsidy(test.height)

		if got != test.expected {
			t.Fatalf(
				"height %d: expected %d, got %d",
				test.height,
				test.expected,
				got,
			)
		}
	}
}

func TestSubsidyEventuallyBecomesZero(t *testing.T) {
	reward := BlockSubsidy(64 * params.HalvingInterval)

	if reward != 0 {
		t.Fatalf("expected zero subsidy, got %d", reward)
	}
}
