package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsOpenCLArtifactWorkflowContract(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "khushi-windows-opencl.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Windows OpenCL workflow: %v", err)
	}
	text := string(data)
	for _, token := range []string{
		"windows-2022",
		"vcpkg install opencl:x64-windows",
		"khushi_miner_opencl.cpp",
		"khushi_pow.cl",
		"khushi-miner-opencl.exe",
		"Get-FileHash",
		"actions/upload-artifact@v4",
		"khushi-miner-opencl-windows",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("Windows OpenCL workflow missing required token %q", token)
		}
	}
}
