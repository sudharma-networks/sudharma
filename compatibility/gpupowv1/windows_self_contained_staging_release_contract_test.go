package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsMinerPackagesContainSameRevisionLocalStagingVerifier(t *testing.T) {
	workflows := []string{
		"khushi-windows-cuda.yml",
		"khushi-windows-opencl.yml",
	}
	for _, name := range workflows {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", ".github", "workflows", name)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			text := string(raw)
			for _, token := range []string{
				"cmd/sudharma-gpupow-staging/**",
				"rpc/mining_staging*.go",
				"scripts/windows/run-local-staging-gate.ps1",
				"actions/setup-go@v5",
				"go-version-file: go.mod",
				"go build -trimpath",
				"dist/sudharma-gpupow-staging.exe",
				"./cmd/sudharma-gpupow-staging",
				"staging_verifier_artifact=sudharma-gpupow-staging.exe",
				"sudharma-gpupow-staging.exe",
				"run-local-staging-gate.ps1",
				"test-khushi-miner.ps1",
			} {
				if !strings.Contains(text, token) {
					t.Fatalf("%s missing self-contained staging token %q", name, token)
				}
			}
		})
	}
}

func TestPublicReleaseRebuildsForLocalStagingVerifierChanges(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "publish-test-mining-release.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read public release workflow: %v", err)
	}
	text := string(raw)
	for _, token := range []string{
		"cmd/sudharma-gpupow-staging/**",
		"rpc/mining_staging*.go",
		"scripts/windows/run-local-staging-gate.ps1",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("public release workflow missing local-staging rebuild trigger %q", token)
		}
	}
}
