package transactions

import "testing"

func TestFeeCalculationAtMaxSupplyDoesNotOverflow(t *testing.T) {
	const maxSupplyAtomic uint64 = 5_100_000_000_000_000_000

	wantTotal := maxSupplyAtomic / 1000
	wantDevelopment := maxSupplyAtomic / 10000
	wantMining := wantTotal - wantDevelopment

	if got := CalculateFee(maxSupplyAtomic); got != wantTotal {
		t.Fatalf("total fee overflow/rounding mismatch: want %d, got %d", wantTotal, got)
	}
	if got := DevelopmentFee(maxSupplyAtomic); got != wantDevelopment {
		t.Fatalf("development fee overflow/rounding mismatch: want %d, got %d", wantDevelopment, got)
	}
	if got := MiningFee(maxSupplyAtomic); got != wantMining {
		t.Fatalf("mining fee overflow/rounding mismatch: want %d, got %d", wantMining, got)
	}
}

func TestFeeDistributionAlwaysConservesTotalFee(t *testing.T) {
	amounts := []uint64{
		1,
		999,
		1000,
		1001,
		9999,
		10000,
		10001,
		1_000_000,
		5_100_000_000_000_000_000,
	}

	for _, amount := range amounts {
		total := CalculateFee(amount)
		development := DevelopmentFee(amount)
		mining := MiningFee(amount)

		if development > total {
			t.Fatalf("amount %d: development fee %d exceeds total fee %d", amount, development, total)
		}
		if mining != total-development {
			t.Fatalf("amount %d: fee split does not conserve total: total=%d development=%d mining=%d", amount, total, development, mining)
		}
	}
}
