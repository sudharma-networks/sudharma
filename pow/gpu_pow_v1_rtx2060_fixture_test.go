package pow

import (
	"encoding/hex"
	"testing"
)

func TestGPUV1RTX2060HardwareFixtureMatchesGoOracle(t *testing.T) {
	const wantProgramSeed = "613684e3f3b42773073fb9c99e71f2933eed301d450866fe9a5a5c0530a769bd"
	const wantDigest = "2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b"
	wantCache := []string{
		"68fae850a5cc8cddba29c7a56913c7340e69ba0d92830144aab66584e01a20e86b919e515046196e7ef9c006150fff8affc13fc252dea4490ef1bb4527adcb6b",
		"25c2c0f117e806e1a832bc4bbcc444043633f5100f9ddf3714988c1b2de377b6e9d6979803b8deca82d5267d53eccc89fa92e21984e535f4e8193881ab309741",
		"a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c",
		"605f70fb5d9b8f1553027dbac7648e70e314ca31e643521da51b78b8b25d2ab35a5842931cda4e39a20efac8290b6d16a890c5a3f867b19260f85ba788bdeebc",
		"44216b2915d6beffc4ca2c8169950f36294462ac53503471c204035586544ae1a1b7dae7aad22d7cc5501578b0b85378b118bea7a0f545d8985c9f426b05cb00",
		"874ec883bf09e6f15d3a54576fe070194925367ff5302438550ac881314e84cd7b1ea8d8cc2f45d54ddaee552996ff856ad27c240ef6b3a769c253c6d0e8cf9b",
		"d4301c92c2addfc7c7983981263e64dafb2d186e147f508ad346b0b002ae4933efebdced686a61f15282bdfc226bb932ced7d5346f296cf8d2c89f38c2adbc59",
		"14d42ce1d735d05d233dccb89532ee7fdbb10acb45d97f2010c04122677b21375a9ddd9dff63010306414d2ecf8c3fb007df86898b2bb55b61c64f19ebffe140",
	}

	programSeed := GPUV1ProgramSeed(0)
	if got := hex.EncodeToString(programSeed[:]); got != wantProgramSeed {
		t.Fatalf("program seed mismatch: got %s want %s", got, wantProgramSeed)
	}

	cache := GPUV1BuildCache(GPUV1EpochSeed(0), len(wantCache))
	if len(cache) != len(wantCache) {
		t.Fatalf("cache length mismatch: got %d want %d", len(cache), len(wantCache))
	}
	for i := range cache {
		if got := hex.EncodeToString(cache[i][:]); got != wantCache[i] {
			t.Fatalf("cache[%d] mismatch: got %s want %s", i, got, wantCache[i])
		}
	}

	digest := gpuV1ReferenceDigest(nil, 0, 0, cache)
	if got := hex.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("digest mismatch: got %s want %s", got, wantDigest)
	}
}
