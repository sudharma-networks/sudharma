package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsHardwareRunnerExecutesProductionMemoryGate(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "windows", "test-khushi-miner.ps1")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Windows hardware runner: %v", err)
	}
	text := string(data)

	for _, want := range []string{
		"Production memory/chunk allocation (--production-memory-self-test)",
		"--production-memory-self-test",
		"hardware-production-memory=passed",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Windows hardware runner missing production memory gate token %q", want)
		}
	}
}
