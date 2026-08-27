package pow

// gpuV1RegisterPermutations builds the deterministic destination/source
// register schedules used by the programmatic GPU mix. The shuffle uses the
// same KISS99 stream as the other GPU-PoW scheduling primitives so independent
// Go, CUDA and OpenCL implementations can reproduce the ordering exactly.
func gpuV1RegisterPermutations(seedLo, seedHi uint32) ([GPUV1NumRegs]uint32, [GPUV1NumRegs]uint32) {
	rng := gpuV1NewKISS99(seedLo, seedHi)

	var dst [GPUV1NumRegs]uint32
	var src [GPUV1NumRegs]uint32
	for i := uint32(0); i < GPUV1NumRegs; i++ {
		dst[i] = i
		src[i] = i
	}

	gpuV1ShuffleRegisters(&rng, &dst)
	gpuV1ShuffleRegisters(&rng, &src)
	return dst, src
}

func gpuV1ShuffleRegisters(rng *gpuV1KISS99, values *[GPUV1NumRegs]uint32) {
	for i := GPUV1NumRegs - 1; i > 0; i-- {
		j := rng.next() % (i + 1)
		values[i], values[j] = values[j], values[i]
	}
}
