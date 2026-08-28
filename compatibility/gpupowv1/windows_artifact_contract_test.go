package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsNVIDIAArtifactWorkflowContract(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "khushi-windows-cuda.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Windows CUDA workflow: %v", err)
	}
	text := string(data)
	required := []string{
		"windows-2022",
		"Jimver/cuda-toolkit@v0.2.36",
		"khushi_miner.cu",
		"-gencode=arch=compute_61,code=sm_61",
		"-gencode=arch=compute_75,code=sm_75",
		"-gencode=arch=compute_86,code=sm_86",
		"-gencode=arch=compute_89,code=compute_89",
		"khushi-miner-nvidia.exe",
		"Get-FileHash",
		"actions/upload-artifact@v4",
		"khushi-miner-nvidia-windows",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("Windows CUDA workflow missing required token %q", token)
		}
	}

	lower := strings.ToLower(text)
	if strings.Contains(lower, "rtx2060") || strings.Contains(lower, "rtx 2060") {
		t.Fatal("Windows miner artifact must be GPU-model neutral; RTX 2060 is a hardware gate, not a build target")
	}
}
