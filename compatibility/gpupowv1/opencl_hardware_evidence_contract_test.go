package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCLHardwareEvidenceContract(t *testing.T) {
	hostPath := filepath.Join("..", "opencl", "khushi_miner_opencl.cpp")
	host, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("read OpenCL host miner: %v", err)
	}
	hostText := string(host)
	for _, token := range []string{
		"CL_PLATFORM_NAME",
		"CL_PLATFORM_VENDOR",
		"CL_DEVICE_VENDOR",
		"CL_DRIVER_VERSION",
		"CL_DEVICE_VERSION",
		"platform_name",
		"platform_vendor",
		"device_vendor",
		"driver_version",
		"device_version",
		"platform=%s",
		"platform_vendor=%s",
		"device_vendor=%s",
		"driver=%s",
		"opencl_version=%s",
	} {
		if !strings.Contains(hostText, token) {
			t.Fatalf("OpenCL host miner missing hardware evidence token %q", token)
		}
	}

	runnerPath := filepath.Join("..", "..", "scripts", "windows", "test-khushi-miner.ps1")
	runner, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("read Windows hardware runner: %v", err)
	}
	runnerText := string(runner)
	for _, token := range []string{
		"Win32_OperatingSystem",
		"Win32_ComputerSystem",
		"windows_caption=",
		"windows_version=",
		"system_manufacturer=",
		"system_model=",
		"powershell_version=",
	} {
		if !strings.Contains(runnerText, token) {
			t.Fatalf("Windows hardware runner missing reproducibility token %q", token)
		}
	}
}
