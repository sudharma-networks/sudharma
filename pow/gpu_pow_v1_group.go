package pow

// gpuV1ProgrammaticGroupDigest executes the deterministic CPU reference path
// for one complete 16-lane GPU-PoW work group and reduces the resulting lane
// states to the canonical eight-word mix digest. This remains an isolated
// reference primitive until cross-implementation vectors and activation gates
// are complete.
func gpuV1ProgrammaticGroupDigest(workSeed uint64, programSeed [32]byte, cache []GPUV1CacheNode) [8]uint32 {
	var lanes [GPUV1NumLanes][GPUV1NumRegs]uint32
	for lane := uint32(0); lane < GPUV1NumLanes; lane++ {
		lanes[lane] = gpuV1ProgrammaticLaneMix(workSeed, lane, programSeed, cache)
	}
	return gpuV1ReduceLanes(lanes)
}
