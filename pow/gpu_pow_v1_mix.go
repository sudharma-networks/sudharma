package pow

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
)

const (
	GPUV1ProgramPeriod  uint64 = 3
	GPUV1NumRegs        uint32 = 32
	GPUV1NumLanes       uint32 = 16
	GPUV1CacheAccesses  uint32 = 11
	GPUV1MathOperations uint32 = 18
	GPUV1DAGRounds      uint32 = 64
	GPUV1L1CacheBytes   uint32 = 16 * 1024
)

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

func gpuV1RotateLeft32(value uint32, amount uint32) uint32 {
	return bits.RotateLeft32(value, int(amount&31))
}

func gpuV1RotateRight32(value uint32, amount uint32) uint32 {
	return bits.RotateLeft32(value, -int(amount&31))
}

func gpuV1MulHi32(a, b uint32) uint32 {
	return uint32((uint64(a) * uint64(b)) >> 32)
}

func gpuV1CLZ32(value uint32) int {
	return bits.LeadingZeros32(value)
}

func gpuV1PopCount32(value uint32) int {
	return bits.OnesCount32(value)
}

// gpuV1FNV1a is the ProgPoW/KAWPOW FNV1a primitive used to initialize RNG
// state and reduce lane results.
func gpuV1FNV1a(a, b uint32) uint32 {
	const prime uint32 = 0x01000193
	return (a ^ b) * prime
}

// gpuV1RandomMath implements the 11-operation KAWPOW 0.9.4 arithmetic family.
// All operations intentionally use uint32 wraparound semantics so Go, CUDA,
// OpenCL and independent verifiers can reproduce the same result.
func gpuV1RandomMath(a, b, selector uint32) uint32 {
	switch selector % 11 {
	case 0:
		return a + b
	case 1:
		return a * b
	case 2:
		return gpuV1MulHi32(a, b)
	case 3:
		if a < b {
			return a
		}
		return b
	case 4:
		return gpuV1RotateLeft32(a, b)
	case 5:
		return gpuV1RotateRight32(a, b)
	case 6:
		return a & b
	case 7:
		return a | b
	case 8:
		return a ^ b
	case 9:
		return uint32(gpuV1CLZ32(a) + gpuV1CLZ32(b))
	default:
		return uint32(gpuV1PopCount32(a) + gpuV1PopCount32(b))
	}
}

// gpuV1RandomMerge implements KAWPOW 0.9.4's four entropy-preserving merge
// operations. Rotation distance is selected from the high selector bits and
// is always in the range 1..31.
func gpuV1RandomMerge(a, b, selector uint32) uint32 {
	x := ((selector >> 16) % 31) + 1
	switch selector % 4 {
	case 0:
		return (a * 33) + b
	case 1:
		return (a ^ b) * 33
	case 2:
		return gpuV1RotateLeft32(a, x) ^ b
	default:
		return gpuV1RotateRight32(a, x) ^ b
	}
}
