package pow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKhushiV022LauncherKeepsConsoleAndTranscriptOnSuccessOrFailure(t *testing.T) {
	root := v022RepoRoot(t)
	launcherPath := filepath.Join(root, filepath.FromSlash("scripts/windows/Run-GPU-Test.bat"))
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatal(err)
	}
	launcher := string(data)
	for _, marker := range []string{
		"Khushi Hardware Test v0.2.2",
		"khushi-hardware-test-console.log",
		"FAIL_REASON",
		"Press any key to close this window.",
		"pause >nul",
	} {
		if !strings.Contains(launcher, marker) {
			t.Errorf("launcher missing persistent-debug marker %q", marker)
		}
	}
	if strings.Contains(strings.ToLower(launcher), "timeout /t") {
		t.Error("launcher must not auto-close on a timeout")
	}
}
