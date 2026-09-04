package pow

import "testing"

func TestGPUV1RegisterPermutationsDeterministic(t *testing.T) {
	dstA, srcA := gpuV1RegisterPermutations(0x11223344, 0xaabbccdd)
	dstB, srcB := gpuV1RegisterPermutations(0x11223344, 0xaabbccdd)
	if dstA != dstB || srcA != srcB {
		t.Fatal("register permutations are not deterministic")
	}
}

func TestGPUV1RegisterPermutationsContainEveryRegister(t *testing.T) {
	dst, src := gpuV1RegisterPermutations(0x11223344, 0xaabbccdd)
	assertGPUV1Permutation(t, dst)
	assertGPUV1Permutation(t, src)
}

func TestGPUV1RegisterPermutationsDependOnSeed(t *testing.T) {
	dstA, srcA := gpuV1RegisterPermutations(0x11223344, 0xaabbccdd)
	dstB, srcB := gpuV1RegisterPermutations(0x11223344, 0xaabbccde)
	if dstA == dstB && srcA == srcB {
		t.Fatal("different seeds produced identical register permutations")
	}
}

func TestGPUV1RegisterPermutationVector(t *testing.T) {
	dst, src := gpuV1RegisterPermutations(0x11223344, 0xaabbccdd)
	wantDst := [GPUV1NumRegs]uint32{10, 24, 7, 8, 6, 11, 29, 0, 20, 14, 25, 22, 23, 19, 16, 17, 4, 13, 28, 27, 12, 30, 3, 18, 9, 15, 31, 21, 2, 1, 26, 5}
	wantSrc := [GPUV1NumRegs]uint32{0, 18, 14, 29, 11, 6, 23, 20, 19, 7, 12, 3, 2, 8, 28, 21, 13, 17, 31, 5, 25, 24, 16, 1, 4, 27, 26, 30, 22, 15, 10, 9}
	if dst != wantDst {
		t.Fatalf("destination permutation mismatch:\n got %v\nwant %v", dst, wantDst)
	}
	if src != wantSrc {
		t.Fatalf("source permutation mismatch:\n got %v\nwant %v", src, wantSrc)
	}
}

func assertGPUV1Permutation(t *testing.T, values [GPUV1NumRegs]uint32) {
	t.Helper()
	var seen [GPUV1NumRegs]bool
	for _, value := range values {
		if value >= GPUV1NumRegs {
			t.Fatalf("register index %d out of range", value)
		}
		if seen[value] {
			t.Fatalf("register index %d appears more than once", value)
		}
		seen[value] = true
	}
	for i, ok := range seen {
		if !ok {
			t.Fatalf("register index %d missing from permutation", i)
		}
	}
}
