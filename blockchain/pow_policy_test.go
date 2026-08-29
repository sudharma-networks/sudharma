package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestPoWPolicyVersionMatrix(t *testing.T) {
	cases := []struct {
		name		string
		activation	uint64
		height		uint64
		version		uint32
		allowed		bool
	}{
		{name: "disabled legacy", activation: params.GPUV1ActivationDisabled, height: 0, version: 1, allowed: true},
		{name: "disabled gpu rejected", activation: params.GPUV1ActivationDisabled, height: 100, version: 2},
		{name: "legacy before", activation: 100, height: 99, version: 1, allowed: true},
		{name: "gpu before rejected", activation: 100, height: 99, version: 2},
		{name: "legacy at rejected", activation: 100, height: 100, version: 1},
		{name: "gpu at", activation: 100, height: 100, version: 2, allowed: true},
		{name: "gpu after", activation: 100, height: 101, version: 2, allowed: true},
		{name: "future version rejected", activation: 100, height: 101, version: 3},
		{name: "zero version rejected", activation: 100, height: 99, version: 0},
		{name: "maximum height gpu", activation: 100, height: ^uint64(0), version: 2, allowed: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := PoWPolicy{GPUV1ActivationHeight: tc.activation}
			if got := policy.VersionAllowed(tc.version, tc.height); got != tc.allowed {
				t.Fatalf("VersionAllowed(%d, %d) = %v, want %v", tc.version, tc.height, got, tc.allowed)
			}
		})
	}
}

func TestPoWPolicyVersionAtHeight(t *testing.T) {
	cases := []struct {
		name	string
		policy	PoWPolicy
		height	uint64
		want	uint32
	}{
		{name: "disabled", policy: LegacyOnlyPoWPolicy(), height: ^uint64(0), want: 1},
		{name: "before", policy: PoWPolicy{GPUV1ActivationHeight: 100}, height: 99, want: 1},
		{name: "at", policy: PoWPolicy{GPUV1ActivationHeight: 100}, height: 100, want: 2},
		{name: "after", policy: PoWPolicy{GPUV1ActivationHeight: 100}, height: 101, want: 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.policy.VersionAtHeight(tc.height)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("VersionAtHeight(%d) = %d, want %d", tc.height, got, tc.want)
			}
		})
	}
}
