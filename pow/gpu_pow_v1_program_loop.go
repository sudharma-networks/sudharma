package pow

import "encoding/binary"

const gpuV1DAGLoads uint32 = 4

// gpuV1ProgrammaticLaneMix executes the deterministic CPU reference path for
// one GPU-PoW lane. It deliberately composes the already-fixed Sudharma v1
// primitives (lane initialization, KISS99 scheduling, register permutations,
// random math/merge, and light-verifiable dataset items) without changing the
// active block-hash dispatcher. This reference path is the oracle that later
// CUDA/OpenCL implementations must match before consensus activation.
func gpuV1ProgrammaticLaneMix(workSeed uint64, lane uint32, programSeed [32]byte, cache []GPUV1CacheNode) [GPUV1NumRegs]uint32 {
	mix := gpuV1InitLane(workSeed, lane)
	if len(cache) == 0 {
		return mix
	}

	seedLo := binary.LittleEndian.Uint32(programSeed[0:4])
	seedHi := binary.LittleEndian.Uint32(programSeed[4:8])
	dstPerm, srcPerm := gpuV1RegisterPermutations(seedLo, seedHi)
	rng := gpuV1NewKISS99(seedLo, seedHi)

	for round := uint32(0); round < GPUV1DAGRounds; round++ {
		// A lane-dependent dataset address keeps the CPU reference light
		// verifiable while preserving the memory-hard DAG access boundary.
		dagIndex := gpuV1FNV(mix[0]^round, lane^rng.next())
		dagItem := GPUV1DatasetItem(cache, dagIndex)

		for load := uint32(0); load < gpuV1DAGLoads; load++ {
			dst := dstPerm[(round*gpuV1DAGLoads+load)%GPUV1NumRegs]
			word := gpuV1Word(dagItem, load)
			mix[dst] = gpuV1RandomMerge(mix[dst], word, rng.next())
		}

		for access := uint32(0); access < GPUV1CacheAccesses; access++ {
			src := srcPerm[(round+access)%GPUV1NumRegs]
			selector := mix[src] ^ rng.next()
			cacheNode := cache[int(selector%uint32(len(cache)))]
			cacheWord := gpuV1Word(cacheNode, (selector>>16)%16)
			dst := dstPerm[(round+gpuV1DAGLoads+access)%GPUV1NumRegs]
			mix[dst] = gpuV1RandomMerge(mix[dst], cacheWord, rng.next())
		}

		for op := uint32(0); op < GPUV1MathOperations; op++ {
			dst := dstPerm[(round+gpuV1DAGLoads+GPUV1CacheAccesses+op)%GPUV1NumRegs]
			srcA := srcPerm[(round+op)%GPUV1NumRegs]
			srcB := srcPerm[(round+op+1)%GPUV1NumRegs]
			value := gpuV1RandomMath(mix[srcA], mix[srcB], rng.next())
			mix[dst] = gpuV1RandomMerge(mix[dst], value, rng.next())
		}
	}

	return mix
}
