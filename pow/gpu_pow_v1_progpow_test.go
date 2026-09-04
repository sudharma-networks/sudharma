package pow

import "testing"

func TestGPUV1CoreParameters(t *testing.T) {
	if GPUV1NumRegs != 32 {
		t.Fatalf("num regs = %d want 32", GPUV1NumRegs)
	}
	if GPUV1NumLanes != 16 {
		t.Fatalf("num lanes = %d want 16", GPUV1NumLanes)
	}
	if GPUV1CacheAccesses != 11 {
		t.Fatalf("cache accesses = %d want 11", GPUV1CacheAccesses)
	}
	if GPUV1MathOperations != 18 {
		t.Fatalf("math operations = %d want 18", GPUV1MathOperations)
	}
	if GPUV1DAGRounds != 64 {
		t.Fatalf("DAG rounds = %d want 64", GPUV1DAGRounds)
	}
	if GPUV1L1CacheBytes != 16*1024 {
		t.Fatalf("L1 cache bytes = %d want %d", GPUV1L1CacheBytes, 16*1024)
	}
}

func TestGPUV1RandomMathContract(t *testing.T) {
	var a uint32 = 0x12345678
	var b uint32 = 0x0f1e2d3c

	cases := []struct {
		selector uint32
		want     uint32
	}{
		{0, a + b},
		{1, a * b},
		{2, gpuV1MulHi32(a, b)},
		{3, b},
		{4, gpuV1RotateLeft32(a, b)},
		{5, gpuV1RotateRight32(a, b)},
		{6, a & b},
		{7, a | b},
		{8, a ^ b},
		{9, uint32(gpuV1CLZ32(a) + gpuV1CLZ32(b))},
		{10, uint32(gpuV1PopCount32(a) + gpuV1PopCount32(b))},
	}

	for _, tc := range cases {
		if got := gpuV1RandomMath(a, b, tc.selector); got != tc.want {
			t.Fatalf("selector %d: got %08x want %08x", tc.selector, got, tc.want)
		}
	}
}

func TestGPUV1RandomMergeContract(t *testing.T) {
	var a uint32 = 0x11223344
	var b uint32 = 0xaabbccdd

	cases := []struct {
		selector uint32
		want     uint32
	}{
		{0, a*33 + b},
		{1, (a ^ b) * 33},
		{2, gpuV1RotateLeft32(a, 1) ^ b},
		{3, gpuV1RotateRight32(a, 1) ^ b},
		{0x001f0002, gpuV1RotateLeft32(a, 1) ^ b},
	}

	for _, tc := range cases {
		if got := gpuV1RandomMerge(a, b, tc.selector); got != tc.want {
			t.Fatalf("selector %08x: got %08x want %08x", tc.selector, got, tc.want)
		}
	}
}
