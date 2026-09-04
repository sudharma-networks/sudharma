package pow

import "encoding/binary"

const gpuV1DatasetParents = 512

// GPUV1DatasetItem derives one deterministic light-verifiable item from the
// epoch cache. The full item is recomputed from cache state; no precomputed
// GPU dataset allocation is required by the CPU reference verifier.
func GPUV1DatasetItem(cache []GPUV1CacheNode, index uint32) GPUV1CacheNode {
	if len(cache) == 0 {
		return GPUV1CacheNode{}
	}

	mix := cache[int(index)%len(cache)]
	first := binary.LittleEndian.Uint32(mix[0:4]) ^ index
	binary.LittleEndian.PutUint32(mix[0:4], first)
	mix = gpuV1Keccak512(mix[:])

	for parent := uint32(0); parent < gpuV1DatasetParents; parent++ {
		selector := gpuV1FNV(index^parent, gpuV1Word(mix, parent%16))
		parentNode := cache[int(selector%uint32(len(cache)))]
		for word := uint32(0); word < 16; word++ {
			mixed := gpuV1FNV(gpuV1Word(mix, word), gpuV1Word(parentNode, word))
			gpuV1PutWord(&mix, word, mixed)
		}
	}

	return gpuV1Keccak512(mix[:])
}

func gpuV1FNV(a, b uint32) uint32 {
	const prime uint32 = 0x01000193
	return (a * prime) ^ b
}

func gpuV1Word(node GPUV1CacheNode, word uint32) uint32 {
	offset := int(word) * 4
	return binary.LittleEndian.Uint32(node[offset : offset+4])
}

func gpuV1PutWord(node *GPUV1CacheNode, word uint32, value uint32) {
	offset := int(word) * 4
	binary.LittleEndian.PutUint32(node[offset:offset+4], value)
}
