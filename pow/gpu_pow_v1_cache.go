package pow

import (
	"encoding/binary"

	"golang.org/x/crypto/sha3"
)

const gpuV1CacheRounds = 3

// GPUV1CacheNode is one 64-byte epoch-cache item. The cache is deliberately
// built from Keccak-512 chained nodes with Ethash-style mixing so later light
// verification and GPU dataset generation can share the same deterministic
// foundation without copying another chain's seed schedule or parameters.
type GPUV1CacheNode [64]byte

// GPUV1BuildCache builds a deterministic cache of nodeCount items from the
// Sudharma-specific epoch seed. nodeCount is parameterized so CI vectors can
// remain small; production cache sizing is defined separately from this core
// generator.
func GPUV1BuildCache(seed [32]byte, nodeCount uint32) []GPUV1CacheNode {
	if nodeCount == 0 {
		return nil
	}

	cache := make([]GPUV1CacheNode, int(nodeCount))
	cache[0] = gpuV1Keccak512(seed[:])
	for i := 1; i < len(cache); i++ {
		cache[i] = gpuV1Keccak512(cache[i-1][:])
	}

	for round := 0; round < gpuV1CacheRounds; round++ {
		for i := 0; i < len(cache); i++ {
			prev := cache[(i-1+len(cache))%len(cache)]
			v := binary.LittleEndian.Uint32(cache[i][0:4]) % nodeCount
			mixed := gpuV1XORNode(prev, cache[int(v)])
			cache[i] = gpuV1Keccak512(mixed[:])
		}
	}

	return cache
}

func gpuV1Keccak512(data []byte) GPUV1CacheNode {
	h := sha3.NewLegacyKeccak512()
	_, _ = h.Write(data)
	sum := h.Sum(nil)
	var out GPUV1CacheNode
	copy(out[:], sum)
	return out
}

func gpuV1XORNode(a, b GPUV1CacheNode) GPUV1CacheNode {
	var out GPUV1CacheNode
	for i := range out {
		out[i] = a[i] ^ b[i]
	}
	return out
}
