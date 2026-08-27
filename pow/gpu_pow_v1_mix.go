package pow

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
)

const GPUV1ProgramPeriod uint64 = 3

var gpuV1ProgramSeedDomain = []byte("SUDHARMA-GPU-POW-V1-PROGRAM-SEED\x00")

// GPUV1ProgramForHeight changes the program every three blocks, matching the
// proven KAWPOW cadence while keeping Sudharma's own deterministic seed
// namespace.
func GPUV1ProgramForHeight(height uint64) uint64 {
	return height / GPUV1ProgramPeriod
}

// GPUV1ProgramSeed derives the deterministic seed used to build the
// programmatic mix schedule for one program period.
func GPUV1ProgramSeed(program uint64) [32]byte {
	input := make([]byte, len(gpuV1ProgramSeedDomain)+8)
	copy(input, gpuV1ProgramSeedDomain)
	binary.BigEndian.PutUint64(input[len(gpuV1ProgramSeedDomain):], program)
	return sha256.Sum256(input)
}

func gpuV1RotateRight32(value uint32, amount uint32) uint32 {
	return bits.RotateLeft32(value, -int(amount&31))
}

// gpuV1FNV1a is the ProgPoW/KAWPOW merge primitive.
func gpuV1FNV1a(a, b uint32) uint32 {
	const prime uint32 = 0x01000193
	return (a ^ b) * prime
}

// gpuV1Merge provides a small deterministic family of integer merge
// operations chosen to map directly to commodity GPU 32-bit ALUs. The mode is
// selected by the program schedule; consensus fixes these semantics.
func gpuV1Merge(dst, src uint32, mode uint32) uint32 {
	switch mode & 3 {
	case 0:
		return gpuV1FNV1a(dst, src)
	case 1:
		rotation := ((src >> 27) & 31) + 1
		return dst ^ gpuV1RotateRight32(src, rotation)
	case 2:
		return dst + src*33
	default:
		rotation := ((src >> 28) & 31) + 1
		return gpuV1RotateRight32(dst, rotation) ^ src
	}
}
