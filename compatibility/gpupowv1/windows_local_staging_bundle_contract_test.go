package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsLocalStagingBundleContract(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "khushi-windows-staging-verifier.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read Windows staging verifier workflow: %v", err)
	}
	workflowText := string(workflow)
	for _, token := range []string{
		"sudharma-gpupow-staging.exe",
		"cmd/sudharma-gpupow-staging",
		"windows-amd64",
		"SHA256SUMS.txt",
		"build-metadata.txt",
		"run-local-staging-gate.ps1",
		"khushi-staging-verifier-windows-amd64",
	} {
		if !strings.Contains(workflowText, token) {
			t.Fatalf("Windows staging verifier workflow missing %q", token)
		}
	}

	scriptPath := filepath.Join("..", "..", "scripts", "windows", "run-local-staging-gate.ps1")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read local staging gate script: %v", err)
	}
	text := string(script)
	for _, token := range []string{
		"127.0.0.1:28646",
		"sudharma-gpupow-staging.exe",
		"test-khushi-miner.ps1",
		"-SubmitStagingSolution",
		"-StagingEndpoint",
		"http://127.0.0.1:28646",
		"SHA256SUMS.txt",
		"Get-FileHash",
		"Stop-Process",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("local staging gate script missing %q", token)
		}
	}
	for lineNumber, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), "\\") {
			t.Fatalf("local staging gate script uses invalid PowerShell backslash continuation on line %d: %q", lineNumber+1, line)
		}
	}
	if strings.Contains(text, "Seed-1") || strings.Contains(text, "Seed-2") || strings.Contains(text, "sudharma.service") {
		t.Fatal("local staging gate must not target seed nodes or the live node service")
	}
}
