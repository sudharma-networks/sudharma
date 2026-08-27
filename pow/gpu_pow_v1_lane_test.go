package pow

import "testing"

func TestGPUV1InitLaneVector(t *testing.T) {
	got := gpuV1InitLane(0x0123456789abcdef, 7)
	want := [GPUV1NumRegs]uint32{
		0xb893953b, 0xb58fa33a, 0x062cb300, 0xf7d7acba,
		0x50a6d91a, 0x687eca74, 0x4618fa6d, 0x0b68032c,
		0x4f92798c, 0x8c091e01, 0xc52f428c, 0x9cb5224d,
		0xbeeac925, 0x3b318e72, 0x47599fd7, 0x39d7a623,
		0xee7778c5, 0x60ffcdc2, 0xa725f5ed, 0xfa36a92a,
		0x8825968d, 0x63c4b5ee, 0x499ca842, 0x62ebaf6b,
		0x7049c19c, 0x79e9c634, 0x5399a819, 0xb64a4725,
		0x0fed6f7b, 0x006d5387, 0x04ec9b4c, 0xb8f7a55d,
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
