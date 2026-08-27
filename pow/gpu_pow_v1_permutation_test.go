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
