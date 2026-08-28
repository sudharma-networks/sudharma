package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKhushiHardwareTestPackageContract(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "windows", "test-khushi-miner.ps1")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read Windows hardware test script: %v", err)
	}
	text := string(script)
	for _, token := range []string{
		"SHA256SUMS.txt",
		"Get-FileHash",
		"--list-devices",
		"--vector-self-test",
		"--benchmark",
		"nvidia-smi",
		"--mine",
		"AllowMining",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("hardware test script missing %q", token)
		}
	}

	docPath := filepath.Join("..", "..", "docs", "khushi-miner.md")
	doc, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read Khushi miner guide: %v", err)
	}
	docText := string(doc)
	for _, token := range []string{
		"Khushi Algorithm",
		"NVIDIA",
		"OpenCL",
		"AMD",
		"RTX 2060",
		"dynamic",
		"hardware interoperability gate",
		"CPU fallback",
	} {
		if !strings.Contains(docText, token) {
			t.Fatalf("Khushi miner guide missing %q", token)
		}
	}
}
