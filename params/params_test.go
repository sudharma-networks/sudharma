package params

import "testing"

func TestMaximumSupplyIsCanonical51BillionSUDH(t *testing.T) {
	const wantSUDH uint64 = 51_000_000_000
	const wantBaseUnits uint64 = 5_100_000_000_000_000_000

	if MaxSupplySUDH != wantSUDH {
		t.Fatalf("MaxSupplySUDH = %d; want %d", MaxSupplySUDH, wantSUDH)
	}
	if MaxSupply != wantBaseUnits {
		t.Fatalf("MaxSupply = %d; want %d", MaxSupply, wantBaseUnits)
	}
}
