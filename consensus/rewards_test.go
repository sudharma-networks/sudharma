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

func TestMonetaryPolicySupplyCaps(t *testing.T) {
	if got := params.MaxSupplyFor(params.MonetaryPolicyMainnet); got != 5_100_000_000_000_000 {
		t.Fatalf("mainnet max supply: expected %d, got %d", uint64(5_100_000_000_000_000), got)
	}

	if got := params.MaxSupplyFor(params.MonetaryPolicyPublicTestnet); got != params.MaxSupply {
		t.Fatalf("testnet max supply changed: expected %d, got %d", params.MaxSupply, got)
	}
}

func TestValidateMonetaryPolicyRejectsUnknown(t *testing.T) {
	if err := params.ValidateMonetaryPolicy(params.MonetaryPolicy(255)); err == nil {
		t.Fatal("expected unknown monetary policy to be rejected")
	}
}
