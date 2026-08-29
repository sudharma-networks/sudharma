package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsHardwareGateRequiresExplicitStagingEndpoint(t *testing.T) {
	script, err := os.ReadFile("../../scripts/windows/test-khushi-miner.ps1")
	if err != nil {
		t.Fatal(err)
	}
	text := string(script)
	for _, want := range []string{
		"[string]$StagingEndpoint",
		"[switch]$SubmitStagingSolution",
		"SubmitStagingSolution requires -StagingEndpoint",
		"network-submission=not-requested",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hardware test script missing staging gate contract %q", want)
		}
	}
	if strings.Contains(text, "[switch]$AllowMining") {
		t.Fatal("hardware test script must not expose legacy -AllowMining without an explicit staging endpoint")
	}
}

func TestHardwareGateDocumentationMatchesMinerCLI(t *testing.T) {
	doc, err := os.ReadFile("../../docs/khushi-miner.md")
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("../../scripts/windows/test-khushi-miner.ps1")
	if err != nil {
		t.Fatal(err)
	}
	cuda, err := os.ReadFile("../cuda/khushi_miner.cu")
	if err != nil {
		t.Fatal(err)
	}

	docText := string(doc)
	for _, want := range []string{
		"Windows prerequisites",
		"-SubmitStagingSolution",
		"-StagingEndpoint",
		"benchmark only by default",
	} {
		if !strings.Contains(docText, want) {
			t.Fatalf("hardware gate documentation missing %q", want)
		}
	}

	cudaText := string(cuda)
	for _, cliFlag := range []string{"--list-devices", "--vector-self-test", "--benchmark", "--telemetry", "--mine"} {
		if !strings.Contains(string(script), cliFlag) {
			t.Fatalf("hardware test script does not exercise documented CLI flag %s", cliFlag)
		}
		if !strings.Contains(cudaText, cliFlag) {
			t.Fatalf("hardware test script references CLI flag %s not present in CUDA miner", cliFlag)
		}
	}
}
