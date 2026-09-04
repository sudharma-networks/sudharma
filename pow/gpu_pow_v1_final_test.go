package pow

import "testing"

func TestGPUV1FinalizeDigestVector(t *testing.T) {
	var headerDigest [32]byte
	for i := range headerDigest {
		headerDigest[i] = byte(i)
	}
	mix := [8]uint32{
		0x11223344, 0x55667788, 0x99aabbcc, 0xddeeff00,
		0x01234567, 0x89abcdef, 0x0badc0de, 0xfeedface,
	}
	want := [32]byte{
		0x17, 0x4b, 0xcb, 0x77, 0x26, 0xe6, 0x51, 0x36,
		0x31, 0x02, 0xf2, 0xbf, 0x8b, 0xea, 0x89, 0x46,
		0x3a, 0x8a, 0xb4, 0x4c, 0x1e, 0x12, 0xa3, 0x46,
		0xa8, 0x75, 0x9f, 0xf5, 0x58, 0xb1, 0x72, 0x76,
	}

	got := gpuV1FinalizeDigest(headerDigest, mix)
	if got != want {
		t.Fatalf("final digest mismatch:\n got %x\nwant %x", got, want)
	}
}

func TestGPUV1FinalizeDigestDependsOnMix(t *testing.T) {
	var headerDigest [32]byte
	a := gpuV1FinalizeDigest(headerDigest, [8]uint32{})
	mix := [8]uint32{}
	mix[7] = 1
	b := gpuV1FinalizeDigest(headerDigest, mix)
	if a == b {
		t.Fatal("different mix digests produced identical final digest")
	}
}

func TestGPUV1FinalizeDigestDependsOnHeader(t *testing.T) {
	var aHeader [32]byte
	bHeader := aHeader
	bHeader[31] = 1
	mix := [8]uint32{1, 2, 3, 4, 5, 6, 7, 8}
	if gpuV1FinalizeDigest(aHeader, mix) == gpuV1FinalizeDigest(bHeader, mix) {
		t.Fatal("different header digests produced identical final digest")
	}
}
