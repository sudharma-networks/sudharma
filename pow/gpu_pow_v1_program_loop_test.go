package pow

import "testing"

func TestGPUV1ProgrammaticLaneMixDeterministic(t *testing.T) {
	cache := GPUV1BuildCache([32]byte{1, 2, 3, 4}, 32)
	programSeed := GPUV1ProgramSeed(7)
	gotA := gpuV1ProgrammaticLaneMix(0x0123456789abcdef, 3, programSeed, cache)
	gotB := gpuV1ProgrammaticLaneMix(0x0123456789abcdef, 3, programSeed, cache)
	if gotA != gotB {
		t.Fatal("programmatic lane mix is not deterministic")
	}
}

func TestGPUV1ProgrammaticLaneMixDependsOnProgram(t *testing.T) {
	cache := GPUV1BuildCache([32]byte{1, 2, 3, 4}, 32)
	a := gpuV1ProgrammaticLaneMix(0x0123456789abcdef, 3, GPUV1ProgramSeed(7), cache)
	b := gpuV1ProgrammaticLaneMix(0x0123456789abcdef, 3, GPUV1ProgramSeed(8), cache)
	if a == b {
		t.Fatal("different program seeds produced identical lane mix")
	}
}

func TestGPUV1ProgrammaticLaneMixDependsOnLane(t *testing.T) {
	cache := GPUV1BuildCache([32]byte{1, 2, 3, 4}, 32)
	programSeed := GPUV1ProgramSeed(7)
	a := gpuV1ProgrammaticLaneMix(0x0123456789abcdef, 3, programSeed, cache)
	b := gpuV1ProgrammaticLaneMix(0x0123456789abcdef, 4, programSeed, cache)
	if a == b {
		t.Fatal("different lanes produced identical lane mix")
	}
}
