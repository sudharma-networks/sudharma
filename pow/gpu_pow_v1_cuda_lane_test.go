package pow

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGPUV1CUDACompatibilityLaneInitMatchesReference(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable CUDA compatibility compile test uses g++")
	}
	gpp, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ unavailable")
	}

	source := filepath.Join("..", "compatibility", "cuda", "gpupow_v1.cu")
	binary := filepath.Join(t.TempDir(), "sudharma-gpupow-cuda")
	build := exec.Command(gpp, "-std=c++17", "-x", "c++", source, "-o", binary)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile CUDA compatibility source as portable C++: %v\n%s", err, output)
	}

	const seed = "0123456789abcdef"
	cases := []struct {
		lane string
		want string
	}{
		{
			lane: "0",
			want: "b3e0dc65,735a987b,7f1f0efb,442f6d50,3bc63e99,4aa0d4d4,3b6a04e7,4773ce1b,31e5b7d0,b23dde4c,03d376f9,c09493ad,d0bc92e6,4146eb88,0a2513ba,605fd89f,6323ef0a,45ee660f,64fbe2a9,ce1c3323,925c3466,58850440,a550fc51,44cd16fb,2e678a5c,c29290c4,adabce43,3591084a,f4fa006c,94b4f3e5,829248bf,48321dec",
		},
		{
			lane: "15",
			want: "fe0adc6c,61f4b943,c11125e0,1626a3c3,25a856c7,72463adf,09a187f7,e5d2a95a,514323e8,6ddb4204,2856016c,dfb1a566,16ddd2cc,37febb94,c8063cbc,7a85045b,8e0522c9,4b45ce42,be154f7d,d3ae4e0d,08fd8d8d,cd5a0ec4,a7802143,c0a8cdc3,99573f40,5935b8ee,019acf6e,44cf36de,84037911,0e93b2a8,c8494c37,3e2c9642",
		},
	}

	for _, tc := range cases {
		cmd := exec.Command(binary, "--lane-init", seed, tc.lane)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("lane %s init command failed: %v\n%s", tc.lane, err, output)
		}
		got := strings.TrimSpace(string(output))
		if got != "lane-mix="+tc.want {
			t.Fatalf("lane %s mismatch:\n got %s\nwant lane-mix=%s", tc.lane, got, tc.want)
		}
	}
}
