package gpupowv1

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"

	"golang.org/x/crypto/sha3"
)

const (
	epochLength    uint64 = 7500
	programPeriod  uint64 = 3
	cacheRounds           = 3
	datasetParents uint32 = 512
	numRegs        uint32 = 32
	numLanes       uint32 = 16
	cacheAccesses  uint32 = 11
	mathOperations uint32 = 18
	dagRounds      uint32 = 64
	dagLoads       uint32 = 4
	fnvOffsetBasis uint32 = 0x811c9dc5
)

var (
	epochSeedDomain       = []byte("SUDHARMA-GPU-POW-V1-EPOCH-SEED\x00")
	programSeedDomain     = []byte("SUDHARMA-GPU-POW-V1-PROGRAM-SEED\x00")
	referenceHeaderDomain = []byte("SUDHARMA-GPU-POW-V1-REFERENCE-HEADER\x00")
	finalDigestDomain     = []byte("SUDHARMA-GPU-POW-V1-FINAL\x00")
)

type cacheNode [64]byte

type kiss99 struct {
	z     uint32
	w     uint32
	jsr   uint32
	jcong uint32
}

// Digest is an implementation-diversity verifier for the pre-activation
// Sudharma GPU-PoW v1 contract. It intentionally does not import or call the
// consensus/reference pow package; interoperability is judged only against
// the published vector corpus.
func Digest(header []byte, nonce, height uint64, cacheNodes uint32) [32]byte {
	if cacheNodes == 0 {
		return [32]byte{}
	}

	cache := buildCache(epochSeed(height/epochLength), cacheNodes)

	input := make([]byte, 0, len(referenceHeaderDomain)+len(header)+8)
	input = append(input, referenceHeaderDomain...)
	input = append(input, header...)
	var nonceBytes [8]byte
	binary.LittleEndian.PutUint64(nonceBytes[:], nonce)
	input = append(input, nonceBytes[:]...)

	headerDigest := sha256.Sum256(input)
	workSeed := binary.LittleEndian.Uint64(headerDigest[:8])
	program := programSeed(height / programPeriod)
	mix := groupDigest(workSeed, program, cache)
	return finalizeDigest(headerDigest, mix)
}

func epochSeed(epoch uint64) [32]byte {
	input := make([]byte, len(epochSeedDomain)+8)
	copy(input, epochSeedDomain)
	binary.BigEndian.PutUint64(input[len(epochSeedDomain):], epoch)
	return sha256.Sum256(input)
}

func programSeed(program uint64) [32]byte {
	input := make([]byte, len(programSeedDomain)+8)
	copy(input, programSeedDomain)
	binary.BigEndian.PutUint64(input[len(programSeedDomain):], program)
	return sha256.Sum256(input)
}

func buildCache(seed [32]byte, nodeCount uint32) []cacheNode {
	cache := make([]cacheNode, int(nodeCount))
	cache[0] = keccak512(seed[:])
	for i := 1; i < len(cache); i++ {
		cache[i] = keccak512(cache[i-1][:])
	}

	for round := 0; round < cacheRounds; round++ {
		for i := 0; i < len(cache); i++ {
			prev := cache[(i-1+len(cache))%len(cache)]
			v := binary.LittleEndian.Uint32(cache[i][0:4]) % nodeCount
			var mixed cacheNode
			for j := range mixed {
				mixed[j] = prev[j] ^ cache[int(v)][j]
			}
			cache[i] = keccak512(mixed[:])
		}
	}
	return cache
}

func keccak512(data []byte) cacheNode {
	h := sha3.NewLegacyKeccak512()
	_, _ = h.Write(data)
	sum := h.Sum(nil)
	var out cacheNode
	copy(out[:], sum)
	return out
}

func datasetItem(cache []cacheNode, index uint32) cacheNode {
	mix := cache[int(index)%len(cache)]
	binary.LittleEndian.PutUint32(mix[0:4], binary.LittleEndian.Uint32(mix[0:4])^index)
	mix = keccak512(mix[:])

	for parent := uint32(0); parent < datasetParents; parent++ {
		selector := fnv1(index^parent, word(mix, parent%16))
		parentNode := cache[int(selector%uint32(len(cache)))]
		for w := uint32(0); w < 16; w++ {
			putWord(&mix, w, fnv1(word(mix, w), word(parentNode, w)))
		}
	}
	return keccak512(mix[:])
}

func word(node cacheNode, index uint32) uint32 {
	offset := int(index) * 4
	return binary.LittleEndian.Uint32(node[offset : offset+4])
}

func putWord(node *cacheNode, index, value uint32) {
	offset := int(index) * 4
	binary.LittleEndian.PutUint32(node[offset:offset+4], value)
}

func fnv1(a, b uint32) uint32 {
	return (a * 0x01000193) ^ b
}

func fnv1a(a, b uint32) uint32 {
	return (a ^ b) * 0x01000193
}

func newKISS99(seedLo, seedHi uint32) kiss99 {
	z := fnv1a(fnvOffsetBasis, seedLo)
	w := fnv1a(z, seedHi)
	jsr := fnv1a(w, seedLo)
	jcong := fnv1a(jsr, seedHi)
	return kiss99{z: z, w: w, jsr: jsr, jcong: jcong}
}

func (r *kiss99) next() uint32 {
	r.z = 36969*(r.z&0xffff) + (r.z >> 16)
	r.w = 18000*(r.w&0xffff) + (r.w >> 16)
	r.jcong = 69069*r.jcong + 1234567
	r.jsr ^= r.jsr << 17
	r.jsr ^= r.jsr >> 13
	r.jsr ^= r.jsr << 5
	return (((r.z << 16) + r.w) ^ r.jcong) + r.jsr
}

func initLane(seed uint64, lane uint32) [numRegs]uint32 {
	seedLo := uint32(seed)
	seedHi := uint32(seed >> 32)
	rng := kiss99{
		z:     fnv1a(fnvOffsetBasis, seedLo),
		w:     fnv1a(fnvOffsetBasis, seedHi),
		jsr:   fnv1a(fnvOffsetBasis, lane),
		jcong: fnv1a(fnvOffsetBasis, lane),
	}
	var mix [numRegs]uint32
	for i := range mix {
		mix[i] = rng.next()
	}
	return mix
}

func registerPermutations(seedLo, seedHi uint32) ([numRegs]uint32, [numRegs]uint32) {
	rng := newKISS99(seedLo, seedHi)
	var dst [numRegs]uint32
	var src [numRegs]uint32
	for i := uint32(0); i < numRegs; i++ {
		dst[i] = i
		src[i] = i
	}
	for i := numRegs - 1; i > 0; i-- {
		j := rng.next() % (i + 1)
		dst[i], dst[j] = dst[j], dst[i]
		j = rng.next() % (i + 1)
		src[i], src[j] = src[j], src[i]
	}
	return dst, src
}

func randomMath(a, b, selector uint32) uint32 {
	switch selector % 11 {
	case 0:
		return a + b
	case 1:
		return a * b
	case 2:
		return uint32((uint64(a) * uint64(b)) >> 32)
	case 3:
		if a < b {
			return a
		}
		return b
	case 4:
		return bits.RotateLeft32(a, int(b&31))
	case 5:
		return bits.RotateLeft32(a, -int(b&31))
	case 6:
		return a & b
	case 7:
		return a | b
	case 8:
		return a ^ b
	case 9:
		return uint32(bits.LeadingZeros32(a) + bits.LeadingZeros32(b))
	default:
		return uint32(bits.OnesCount32(a) + bits.OnesCount32(b))
	}
}

func randomMerge(a, b, selector uint32) uint32 {
	x := ((selector >> 16) % 31) + 1
	switch selector % 4 {
	case 0:
		return (a * 33) + b
	case 1:
		return (a ^ b) * 33
	case 2:
		return bits.RotateLeft32(a, int(x)) ^ b
	default:
		return bits.RotateLeft32(a, -int(x)) ^ b
	}
}

func laneMix(workSeed uint64, lane uint32, program [32]byte, cache []cacheNode) [numRegs]uint32 {
	mix := initLane(workSeed, lane)
	seedLo := binary.LittleEndian.Uint32(program[0:4])
	seedHi := binary.LittleEndian.Uint32(program[4:8])
	dstPerm, srcPerm := registerPermutations(seedLo, seedHi)
	rng := newKISS99(seedLo, seedHi)

	for round := uint32(0); round < dagRounds; round++ {
		dagIndex := fnv1(mix[0]^round, lane^rng.next())
		dagItem := datasetItem(cache, dagIndex)

		for load := uint32(0); load < dagLoads; load++ {
			dst := dstPerm[(round*dagLoads+load)%numRegs]
			mix[dst] = randomMerge(mix[dst], word(dagItem, load), rng.next())
		}

		for access := uint32(0); access < cacheAccesses; access++ {
			src := srcPerm[(round+access)%numRegs]
			selector := mix[src] ^ rng.next()
			cacheNode := cache[int(selector%uint32(len(cache)))]
			cacheWord := word(cacheNode, (selector>>16)%16)
			dst := dstPerm[(round+dagLoads+access)%numRegs]
			mix[dst] = randomMerge(mix[dst], cacheWord, rng.next())
		}

		for op := uint32(0); op < mathOperations; op++ {
			dst := dstPerm[(round+dagLoads+cacheAccesses+op)%numRegs]
			srcA := srcPerm[(round+op)%numRegs]
			srcB := srcPerm[(round+op+1)%numRegs]
			value := randomMath(mix[srcA], mix[srcB], rng.next())
			mix[dst] = randomMerge(mix[dst], value, rng.next())
		}
	}
	return mix
}

func reduceLane(lane [numRegs]uint32) uint32 {
	digest := fnvOffsetBasis
	for _, value := range lane {
		digest = fnv1a(digest, value)
	}
	return digest
}

func groupDigest(workSeed uint64, program [32]byte, cache []cacheNode) [8]uint32 {
	var digest [8]uint32
	for i := range digest {
		digest[i] = fnvOffsetBasis
	}
	for lane := uint32(0); lane < numLanes; lane++ {
		laneDigest := reduceLane(laneMix(workSeed, lane, program, cache))
		index := lane % uint32(len(digest))
		digest[index] = fnv1a(digest[index], laneDigest)
	}
	return digest
}

func finalizeDigest(headerDigest [32]byte, mix [8]uint32) [32]byte {
	input := make([]byte, 0, len(finalDigestDomain)+32+8*4)
	input = append(input, finalDigestDomain...)
	input = append(input, headerDigest[:]...)
	var encoded [4]byte
	for _, value := range mix {
		binary.LittleEndian.PutUint32(encoded[:], value)
		input = append(input, encoded[:]...)
	}
	return sha256.Sum256(input)
}
