package pow

// gpuV1ReduceLane compresses one lane's register state to a single 32-bit
// digest using the ProgPoW/KAWPOW FNV1a reduction pattern.
func gpuV1ReduceLane(lane [GPUV1NumRegs]uint32) uint32 {
	digest := gpuV1FNVOffsetBasis
	for _, value := range lane {
		digest = gpuV1FNV1a(digest, value)
	}
	return digest
}

// gpuV1ReduceLanes compresses the 16 per-lane digests into the canonical
// eight-word mix digest. Lanes are folded modulo eight so CUDA/OpenCL and the
// Go light verifier share an identical reduction contract.
func gpuV1ReduceLanes(lanes [GPUV1NumLanes][GPUV1NumRegs]uint32) [8]uint32 {
	var digest [8]uint32
	for i := range digest {
		digest[i] = gpuV1FNVOffsetBasis
	}
	for lane := uint32(0); lane < GPUV1NumLanes; lane++ {
		word := lane % uint32(len(digest))
		digest[word] = gpuV1FNV1a(digest[word], gpuV1ReduceLane(lanes[lane]))
	}
	return digest
}
