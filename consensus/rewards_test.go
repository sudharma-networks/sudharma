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

func TestMainnetEmissionTableInvariants(t *testing.T) {
	if len(params.MainnetEmissionEpochs) != 40 {
		t.Fatalf("expected 40 epochs, got %d", len(params.MainnetEmissionEpochs))
	}

	var total uint64
	for _, epoch := range params.MainnetEmissionEpochs {
		total += epoch.Issuance
	}
	if total != params.MainnetMaxSupply {
		t.Fatalf("expected exact 51M issuance, got %d base units", total)
	}

	yearTargets := []uint64{
		8_160_000, 7_140_000, 6_630_000, 6_120_000, 5_610_000,
		5_100_000, 4_080_000, 3_570_000, 2_550_000, 2_040_000,
	}
	for year, wantSUDH := range yearTargets {
		var got uint64
		for q := 0; q < 4; q++ {
			got += params.MainnetEmissionEpochs[year*4+q].Issuance
		}
		want := wantSUDH * params.CoinDecimals
		if got != want {
			t.Fatalf("year %d: expected %d, got %d", year+1, want, got)
		}
	}
}
