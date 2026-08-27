package pow

import "testing"

func TestGPUV1InteroperabilityVectorProbe(t *testing.T) {
	cases := []struct {
		name   string
		header string
		nonce  uint64
		height uint64
	}{
		{name: "genesis-program-zero", header: "", nonce: 0, height: 0},
		{name: "program-zero-max-nonce", header: "interop-a", nonce: ^uint64(0), height: 2},
		{name: "program-one-boundary", header: "interop-b", nonce: 1, height: 3},
		{name: "epoch-zero-tail", header: "interop-c", nonce: 0x0102030405060708, height: GPUV1EpochLength - 1},
		{name: "epoch-one-head", header: "interop-d", nonce: 0x8877665544332211, height: GPUV1EpochLength},
		{name: "locked-reference", header: "sudharma-gpu-pow-v1-reference-header", nonce: 0x0123456789abcdef, height: 22501},
	}

	for _, tc := range cases {
		cache := GPUV1BuildCache(GPUV1EpochSeed(GPUV1EpochForHeight(tc.height)), 8)
		digest := gpuV1ReferenceDigest([]byte(tc.header), tc.nonce, tc.height, cache)
		t.Errorf("VECTOR %s header=%x nonce=%016x height=%d cache_nodes=8 digest=%x", tc.name, []byte(tc.header), tc.nonce, tc.height, digest)
	}
}
