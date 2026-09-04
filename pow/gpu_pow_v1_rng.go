package pow

const gpuV1FNVOffsetBasis uint32 = 0x811c9dc5

// gpuV1KISS99 is the 32-bit KISS99 generator used by ProgPoW/KAWPOW to
// deterministically schedule register, cache and arithmetic operations.
type gpuV1KISS99 struct {
	z     uint32
	w     uint32
	jsr   uint32
	jcong uint32
}

// gpuV1NewKISS99 derives the KISS99 state from two 32-bit program seed words
// using the KAWPOW FNV1a initialization sequence.
func gpuV1NewKISS99(seedLo, seedHi uint32) gpuV1KISS99 {
	z := gpuV1FNV1a(gpuV1FNVOffsetBasis, seedLo)
	w := gpuV1FNV1a(z, seedHi)
	jsr := gpuV1FNV1a(w, seedLo)
	jcong := gpuV1FNV1a(jsr, seedHi)
	return gpuV1KISS99{z: z, w: w, jsr: jsr, jcong: jcong}
}

func (r *gpuV1KISS99) next() uint32 {
	r.z = 36969*(r.z&0xffff) + (r.z >> 16)
	r.w = 18000*(r.w&0xffff) + (r.w >> 16)

	r.jcong = 69069*r.jcong + 1234567

	r.jsr ^= r.jsr << 17
	r.jsr ^= r.jsr >> 13
	r.jsr ^= r.jsr << 5

	return (((r.z << 16) + r.w) ^ r.jcong) + r.jsr
}
