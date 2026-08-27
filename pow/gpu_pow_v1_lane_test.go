package pow

import "testing"

func TestGPUV1InitLaneVector(t *testing.T) {
	got := gpuV1InitLane(0x0123456789abcdef, 7)
	want := [GPUV1NumRegs]uint32{
		0xb893953b, 0xb58fa33a, 0x062cb300, 0xf7d7acba,
		0x50a6d91a, 0x687eca74, 0x4618fa6d, 0x0b68032c,
		0x4f91e5cc, 0x8c0ab6c1, 0xc52e48cc, 0x9cb3550d,
		0xbee904e5, 0x3b315072, 0x475acdd7, 0x39d6f223,
		0xee76e405, 0x60fe26c2, 0xa7254a6d, 0xfa35046a,
		0x8823024d, 0x63c45b2e, 0x499bb182, 0x62ec846b,
		0x704787dc, 0x79e74eb4, 0x539a1119, 0xb64a6ee5,
		0x0fed177b, 0x006d53b7, 0x04ec984c, 0xb8f776dd,
	}
	if got != want {
		t.Fatalf("lane vector mismatch:\n got %08x\nwant %08x", got, want)
	}
}

func TestGPUV1InitLaneDependsOnLane(t *testing.T) {
	a := gpuV1InitLane(0x0123456789abcdef, 0)
	b := gpuV1InitLane(0x0123456789abcdef, 1)
	if a == b {
		t.Fatal("different lanes produced identical initial register state")
	}
}

func TestGPUV1InitLaneDeterministic(t *testing.T) {
	a := gpuV1InitLane(0x0123456789abcdef, 15)
	b := gpuV1InitLane(0x0123456789abcdef, 15)
	if a != b {
		t.Fatal("lane initialization is not deterministic")
	}
}
