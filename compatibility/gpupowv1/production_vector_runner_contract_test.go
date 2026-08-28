package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsHardwareRunnerExecutesProductionVectorGate(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "windows", "test-khushi-miner.ps1")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Windows hardware runner: %v", err)
	}
	text := string(data)
	for _, token := range []string{
		"ProductionVectorPath",
		"khushi-production-vectors-nvidia.exe",
		"khushi-production-vectors-opencl.exe",
		"production-vector-sha256=ok",
		"Production dataset boundary vectors",
		"& $ProductionVectorPath --device $Device",
		"hardware-production-vectors=passed",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("Windows hardware runner missing production vector token %q", token)
		}
	}
}
