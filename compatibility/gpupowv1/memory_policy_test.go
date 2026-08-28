package gpupowv1

import "testing"

func TestProductionMemoryPolicyFitsFourGiBTarget(t *testing.T) {
	p := GPUV1ProductionMemory
	if p.DatasetBytes != 2<<30 || p.CacheBytes != 16<<20 ||
		p.RuntimeReserveBytes != 256<<20 || p.ChunkBytes != 256<<20 ||
		p.ItemBytes != 64 || p.MinimumDedicatedVRAMBytes != 4<<30 {
		t.Fatalf("unexpected policy: %+v", p)
	}

	required, ok := p.RequiredVRAMBytes()
	if !ok || required != 2432696320 {
		t.Fatalf("required=%d ok=%v", required, ok)
	}
	if !p.EligibleDedicatedVRAM(4<<30, required) {
		t.Fatal("conforming 4 GiB device rejected")
	}
	if p.EligibleDedicatedVRAM((4<<30)-1, required) {
		t.Fatal("sub-4 GiB device accepted")
	}
}
