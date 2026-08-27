package pow

import "testing"

func TestGPUV1ProgramPeriod(t *testing.T) {
	if GPUV1ProgramPeriod != 3 {
		t.Fatalf("program period = %d want 3", GPUV1ProgramPeriod)
	}
	if got := GPUV1ProgramForHeight(0); got != 0 {
		t.Fatalf("height 0 program = %d", got)
	}
	if got := GPUV1ProgramForHeight(2); got != 0 {
		t.Fatalf("height 2 program = %d", got)
	}
	if got := GPUV1ProgramForHeight(3); got != 1 {
		t.Fatalf("height 3 program = %d", got)
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

func TestGPUV1FNV1a(t *testing.T) {
	var a uint32 = 0x11223344
	var b uint32 = 0xaabbccdd
	var prime uint32 = 0x01000193
	want := (a ^ b) * prime
	if got := gpuV1FNV1a(a, b); got != want {
		t.Fatalf("fnv1a = %08x want %08x", got, want)
	}
}
