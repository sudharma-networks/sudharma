package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsMinerArtifactsPackageHardwareRunner(t *testing.T) {
	for _, workflow := range []string{"khushi-windows-cuda.yml", "khushi-windows-opencl.yml"} {
		path := filepath.Join("..", "..", ".github", "workflows", workflow)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", workflow, err)
		}
		text := string(data)
		for _, token := range []string{
			"scripts/windows/test-khushi-miner.ps1",
			"dist/test-khushi-miner.ps1",
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("%s missing hardware-runner packaging token %q", workflow, token)
			}
		}
	}
}
