package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionVectorReleaseAssetContract(t *testing.T) {
	releasePath := filepath.Join("..", "..", ".github", "workflows", "publish-test-mining-release.yml")
	releaseData, err := os.ReadFile(releasePath)
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	release := string(releaseData)
	for _, token := range []string{
		"test -f dist/nvidia/khushi-production-vectors-nvidia.exe",
		"test -f dist/nvidia/gpu-pow-v1-production-memory-vectors.json",
		"test -f dist/opencl/khushi-production-vectors-opencl.exe",
		"test -f dist/opencl/gpupow_v1_production_vectors.cl",
		"test -f dist/opencl/gpu-pow-v1-production-memory-vectors.json",
	} {
		if !strings.Contains(release, token) {
			t.Fatalf("release workflow missing production vector asset check %q", token)
		}
	}

	openclPath := filepath.Join("..", "..", ".github", "workflows", "khushi-windows-opencl.yml")
	openclData, err := os.ReadFile(openclPath)
	if err != nil {
		t.Fatalf("read OpenCL workflow: %v", err)
	}
	opencl := string(openclData)
	for _, token := range []string{
		"gpu-pow-v1-production-memory-vectors.json",
		"production_vectors=docs/gpu-pow-v1-production-memory-vectors.json",
	} {
		if !strings.Contains(opencl, token) {
			t.Fatalf("OpenCL workflow missing production vector fixture token %q", token)
		}
	}
}
