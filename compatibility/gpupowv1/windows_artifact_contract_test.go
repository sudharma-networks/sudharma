package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsRTX2060ArtifactWorkflowContract(t *testing.T) {
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
		"-arch=sm_75",
		"khushi-miner-rtx2060.exe",
		"Get-FileHash",
		"actions/upload-artifact@v4",
		"khushi-miner-rtx2060-windows",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("Windows CUDA workflow missing required token %q", token)
		}
	}
}
