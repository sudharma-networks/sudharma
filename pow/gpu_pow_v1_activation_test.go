package pow

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestGPUV1VersionAllowedAtActivationBoundary(t *testing.T) {
	const activationHeight uint64 = 100

	cases := []struct {
		name    string
		height  uint64
		version uint32
		want    bool
	}{
		{name: "legacy-before", height: activationHeight - 1, version: 1, want: true},
		{name: "v2-before-rejected", height: activationHeight - 1, version: 2, want: false},
		{name: "legacy-at-rejected", height: activationHeight, version: 1, want: false},
		{name: "v2-at-accepted", height: activationHeight, version: 2, want: true},
		{name: "legacy-after-rejected", height: activationHeight + 1, version: 1, want: false},
		{name: "v2-after-accepted", height: activationHeight + 1, version: 2, want: true},
		{name: "future-version-rejected", height: activationHeight + 1, version: 3, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GPUV1VersionAllowedAtHeight(tc.version, tc.height, activationHeight); got != tc.want {
				t.Fatalf("version=%d height=%d activation=%d: got %v want %v", tc.version, tc.height, activationHeight, got, tc.want)
			}
		})
	}
}

func TestGPUV1DisabledActivationKeepsLegacyOnly(t *testing.T) {
	for _, height := range []uint64{0, 1, 100, 1_000_000, ^uint64(0) - 1} {
		if !GPUV1VersionAllowedAtHeight(1, height, params.GPUV1ActivationDisabled) {
			t.Fatalf("legacy version rejected while activation disabled at height %d", height)
		}
		if GPUV1VersionAllowedAtHeight(2, height, params.GPUV1ActivationDisabled) {
			t.Fatalf("v2 accepted while activation disabled at height %d", height)
		}
	}
}

func TestGPUV1NetworkActivationDefaultsRemainDisabled(t *testing.T) {
	if params.GPUV1TestnetActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("testnet GPU-PoW activation must remain unarmed before staged deployment gate: got %d", params.GPUV1TestnetActivationHeight)
	}
	if params.GPUV1MainnetActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatalf("mainnet GPU-PoW activation must remain disabled: got %d", params.GPUV1MainnetActivationHeight)
	}
}
