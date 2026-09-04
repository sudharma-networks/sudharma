package pow

import "testing"

func TestGPUV1ProgrammaticGroupDigestMatchesLaneReference(t *testing.T) {
	seed := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	cacheSeed := GPUV1EpochSeed(0)
	cache := GPUV1BuildCache(cacheSeed, 8)
	workSeed := uint64(0x0123456789abcdef)

	var lanes [GPUV1NumLanes][GPUV1NumRegs]uint32
	for lane := uint32(0); lane < GPUV1NumLanes; lane++ {
		lanes[lane] = gpuV1ProgrammaticLaneMix(workSeed, lane, seed, cache)
	}
	want := gpuV1ReduceLanes(lanes)

	got := gpuV1ProgrammaticGroupDigest(workSeed, seed, cache)
	if got != want {
		t.Fatalf("group digest mismatch:\n got %08x\nwant %08x", got, want)
	}
}

func TestGPUV1ProgrammaticGroupDigestDeterministic(t *testing.T) {
	seed := GPUV1ProgramSeed(7)
	cache := GPUV1BuildCache(GPUV1EpochSeed(0), 8)
	a := gpuV1ProgrammaticGroupDigest(0x1122334455667788, seed, cache)
	b := gpuV1ProgrammaticGroupDigest(0x1122334455667788, seed, cache)
	if a != b {
		t.Fatal("group digest is not deterministic")
	}
}

func TestGPUV1ProgrammaticGroupDigestDependsOnWorkSeed(t *testing.T) {
	seed := GPUV1ProgramSeed(7)
	cache := GPUV1BuildCache(GPUV1EpochSeed(0), 8)
	a := gpuV1ProgrammaticGroupDigest(0x1122334455667788, seed, cache)
	b := gpuV1ProgrammaticGroupDigest(0x1122334455667789, seed, cache)
	if a == b {
		t.Fatal("different work seeds produced identical group digests")
	}
}
