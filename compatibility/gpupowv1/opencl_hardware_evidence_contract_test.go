package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsHardwareEvidenceProvenanceContract(t *testing.T) {
	runnerPath := filepath.Join("..", "..", "scripts", "windows", "test-khushi-miner.ps1")
	runner, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("read Windows hardware runner: %v", err)
	}
	runnerText := string(runner)
	for _, token := range []string{
		"Win32_OperatingSystem",
		"Win32_ComputerSystem",
		"Win32_VideoController",
		"windows_caption=",
		"windows_version=",
		"windows_build=",
		"system_manufacturer=",
		"system_model=",
		"computer_name=",
		"powershell_version=",
		"video_name=",
		"video_vendor=",
		"video_driver_version=",
		"video_adapter_ram_bytes=",
	} {
		if !strings.Contains(runnerText, token) {
			t.Fatalf("Windows hardware runner missing reproducibility token %q", token)
		}
	}
}
