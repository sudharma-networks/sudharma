package pow

// gpuV1RegisterPermutations builds the deterministic destination/source
// register schedules used by the programmatic GPU mix. The destination and
// source shuffles consume the KISS99 stream in an interleaved Fisher-Yates
// sequence, matching the proven ProgPoW/KAWPOW scheduling pattern while using
// Sudharma's own program seed namespace.
func gpuV1RegisterPermutations(seedLo, seedHi uint32) ([GPUV1NumRegs]uint32, [GPUV1NumRegs]uint32) {
	rng := gpuV1NewKISS99(seedLo, seedHi)

	var dst [GPUV1NumRegs]uint32
	var src [GPUV1NumRegs]uint32
	for i := uint32(0); i < GPUV1NumRegs; i++ {
		dst[i] = i
		src[i] = i
	}

	for i := GPUV1NumRegs - 1; i > 0; i-- {
		j := rng.next() % (i + 1)
		dst[i], dst[j] = dst[j], dst[i]

		j = rng.next() % (i + 1)
		src[i], src[j] = src[j], src[i]
	}

	return dst, src
}
