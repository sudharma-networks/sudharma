package params

import "testing"

func TestMaximumSupplyIsFiftyOneBillionSUDH(t *testing.T) {
	const wantCoins uint64 = 51_000_000_000
	const wantBaseUnits uint64 = 5_100_000_000_000_000_000

	if MaxSupplySUDH != wantCoins {
		t.Fatalf("maximum supply coins = %d, want %d", MaxSupplySUDH, wantCoins)
	}
	if MaxSupply != wantBaseUnits {
		t.Fatalf("maximum supply base units = %d, want %d", MaxSupply, wantBaseUnits)
	}
	if MaxSupply/CoinDecimals != MaxSupplySUDH {
		t.Fatalf("maximum supply conversion is inconsistent: base=%d decimals=%d coins=%d", MaxSupply, CoinDecimals, MaxSupplySUDH)
	}
}
