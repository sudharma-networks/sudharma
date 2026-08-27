package pow

import "testing"

func TestGPUV1ProgramPeriod(t *testing.T) {
	if GPUV1ProgramPeriod != 10 {
		t.Fatalf("program period = %d want 10", GPUV1ProgramPeriod)
	}
	if got := GPUV1ProgramForHeight(0); got != 0 {
		t.Fatalf("height 0 program = %d", got)
	}
	if got := GPUV1ProgramForHeight(9); got != 0 {
		t.Fatalf("height 9 program = %d", got)
	}
	if got := GPUV1ProgramForHeight(10); got != 1 {
		t.Fatalf("height 10 program = %d", got)
	}
}

func TestGPUV1ProgramSeedDeterministic(t *testing.T) {
	a := GPUV1ProgramSeed(12345)
	b := GPUV1ProgramSeed(12345)
	c := GPUV1ProgramSeed(12346)
	if a != b {
		t.Fatal("program seed is not deterministic")
	}
	if a == c {
		t.Fatal("adjacent program periods share a seed")
	}
	if a == [32]byte{} {
		t.Fatal("program seed must be domain-separated")
	}
}

func TestGPUV1RotateRight32(t *testing.T) {
	if got := gpuV1RotateRight32(0x12345678, 8); got != 0x78123456 {
		t.Fatalf("rotate = %08x", got)
	}
	if got := gpuV1RotateRight32(0x12345678, 0); got != 0x12345678 {
		t.Fatalf("zero rotate = %08x", got)
	}
}

func TestGPUV1MergePrimitive(t *testing.T) {
	if got := gpuV1Merge(0x11223344, 0xaabbccdd, 0); got != gpuV1FNV(0x11223344, 0xaabbccdd) {
		t.Fatalf("merge mode 0 = %08x", got)
	}
	if got := gpuV1Merge(0x11223344, 0xaabbccdd, 1); got == 0x11223344 {
		t.Fatal("merge mode 1 did not alter destination")
	}
}
