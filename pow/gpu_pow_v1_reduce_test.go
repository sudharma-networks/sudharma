package pow

import "testing"

func TestGPUV1ReduceLaneVector(t *testing.T) {
	lane := gpuV1InitLane(0x0123456789abcdef, 7)
	if got := gpuV1ReduceLane(lane); got != 0xad7cb30b {
		t.Fatalf("lane digest = %08x want ad7cb30b", got)
	}
}

func TestGPUV1ReduceLanesVector(t *testing.T) {
	var lanes [GPUV1NumLanes][GPUV1NumRegs]uint32
	for lane := uint32(0); lane < GPUV1NumLanes; lane++ {
		lanes[lane] = gpuV1InitLane(0x0123456789abcdef, lane)
	}

	got := gpuV1ReduceLanes(lanes)
	want := [8]uint32{
		0xcfb8c60e, 0xdf8e5bce, 0x0d4815dc, 0xed15b0f4,
		0x3460559e, 0x296f7424, 0x8e35769e, 0xc2b778d9,
	}
	if got != want {
		t.Fatalf("lane reduction mismatch:\n got %08x\nwant %08x", got, want)
	}
}
