package pow

// gpuV1InitLane expands one 64-bit work seed and lane identifier into the
// per-lane register state used by the programmatic GPU mix. This follows the
// proven ProgPoW/KAWPOW fill-mix construction while keeping Sudharma's own
// header, epoch and program-seed domains.
func gpuV1InitLane(seed uint64, lane uint32) [GPUV1NumRegs]uint32 {
	seedLo := uint32(seed)
	seedHi := uint32(seed >> 32)

	rng := gpuV1KISS99{
		z:     gpuV1FNV1a(gpuV1FNVOffsetBasis, seedLo),
		w:     gpuV1FNV1a(gpuV1FNVOffsetBasis, seedHi),
		jsr:   gpuV1FNV1a(gpuV1FNVOffsetBasis, lane),
		jcong: gpuV1FNV1a(gpuV1FNVOffsetBasis, lane),
	}

	var mix [GPUV1NumRegs]uint32
	for i := range mix {
		mix[i] = rng.next()
	}
	return mix
}
