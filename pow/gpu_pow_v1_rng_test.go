package pow

import "testing"

func TestGPUV1KISS99ReferenceVector(t *testing.T) {
	rng := gpuV1KISS99{
		z:     362436069,
		w:     521288629,
		jsr:   123456789,
		jcong: 380116160,
	}

	want := []uint32{769445856, 742012328, 2121196314, 2805620942, 3214428071}
	for i, expected := range want {
		if got := rng.next(); got != expected {
			t.Fatalf("output %d = %d want %d", i, got, expected)
		}
	}
}

func TestGPUV1KISS99SeedFromProgramWords(t *testing.T) {
	rngA := gpuV1NewKISS99(0x11223344, 0xaabbccdd)
	rngB := gpuV1NewKISS99(0x11223344, 0xaabbccdd)
	rngC := gpuV1NewKISS99(0x11223344, 0xaabbccde)

	for i := 0; i < 8; i++ {
		if a, b := rngA.next(), rngB.next(); a != b {
			t.Fatalf("deterministic stream diverged at %d: %08x != %08x", i, a, b)
		}
	}
	if rngA.next() == rngC.next() {
		t.Fatal("different seed words unexpectedly produced same stream value")
	}
}
