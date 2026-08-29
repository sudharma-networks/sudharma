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

func TestProductionMemoryPolicyRejectsInvalidLayouts(t *testing.T) {
	max := ^uint64(0)
	cases := []struct {
		name   string
		policy ProductionMemoryPolicy
	}{
		{name: "sum overflow", policy: ProductionMemoryPolicy{DatasetBytes: max, CacheBytes: 1, RuntimeReserveBytes: 1, ChunkBytes: 64, ItemBytes: 64, MinimumDedicatedVRAMBytes: 4 << 30}},
		{name: "zero item", policy: ProductionMemoryPolicy{DatasetBytes: 1024, CacheBytes: 64, RuntimeReserveBytes: 64, ChunkBytes: 256, ItemBytes: 0, MinimumDedicatedVRAMBytes: 4 << 30}},
		{name: "item does not divide dataset", policy: ProductionMemoryPolicy{DatasetBytes: 1025, CacheBytes: 64, RuntimeReserveBytes: 64, ChunkBytes: 256, ItemBytes: 64, MinimumDedicatedVRAMBytes: 4 << 30}},
		{name: "item does not divide chunk", policy: ProductionMemoryPolicy{DatasetBytes: 1024, CacheBytes: 64, RuntimeReserveBytes: 64, ChunkBytes: 255, ItemBytes: 64, MinimumDedicatedVRAMBytes: 4 << 30}},
		{name: "chunk does not divide dataset", policy: ProductionMemoryPolicy{DatasetBytes: 1024, CacheBytes: 64, RuntimeReserveBytes: 64, ChunkBytes: 384, ItemBytes: 64, MinimumDedicatedVRAMBytes: 4 << 30}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.policy.Validate(); err == nil {
				t.Fatal("invalid memory policy accepted")
			}
		})
	}

	if err := GPUV1ProductionMemory.Validate(); err != nil {
		t.Fatalf("production memory policy rejected: %v", err)
	}
}
