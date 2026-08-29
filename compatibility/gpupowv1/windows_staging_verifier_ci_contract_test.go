package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsStagingVerifierWorkflowTracksGuideAndParsesPackagedScripts(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "khushi-windows-staging-verifier.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Windows staging verifier workflow: %v", err)
	}
	text := string(raw)
	for _, token := range []string{
		"docs/test-mining/PUBLIC_TEST_MINING.md",
		"Validate packaged PowerShell syntax",
		"[System.Management.Automation.Language.Parser]::ParseFile",
		"scripts/windows/run-local-staging-gate.ps1",
		"scripts/windows/test-khushi-miner.ps1",
		"PowerShell syntax validation failed",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("Windows staging verifier workflow missing %q", token)
		}
	}
}
