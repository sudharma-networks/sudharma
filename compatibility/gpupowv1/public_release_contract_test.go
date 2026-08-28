package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicReleaseTracksHardwareGateAndSameRevisionArtifacts(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "publish-test-mining-release.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read public release workflow: %v", err)
	}
	text := string(workflow)

	for _, token := range []string{
		"scripts/windows/test-khushi-miner.ps1",
		".github/workflows/khushi-windows-cuda.yml",
		".github/workflows/khushi-windows-opencl.yml",
		"headSha",
		"$GITHUB_SHA",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("public release workflow missing freshness contract %q", token)
		}
	}

	if strings.Contains(text, "--status success --limit 1 --json databaseId --jq '.[0].databaseId'") {
		t.Fatal("public release must not select an arbitrary latest successful miner build without binding it to the release commit")
	}
}

func TestPublicReleaseTagTracksArtifactSourceRevision(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "publish-test-mining-release.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read public release workflow: %v", err)
	}
	text := string(workflow)

	for _, token := range []string{
		"git/refs/tags/$TAG",
		"sha=\"$GITHUB_SHA\"",
		"force=true",
		"release_tag_sha",
		"release tag does not match artifact source revision",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("public release workflow missing tag/source revision alignment token %q", token)
		}
	}
}
