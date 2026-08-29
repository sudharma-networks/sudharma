package operations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestGPUActivationAbortPreservesRecordAndManifestBeforeBoundary(t *testing.T) {
	directory := t.TempDir()
	evidenceDirectory := filepath.Join(directory, "abort-evidence")
	record := writeAbortFixture(t, directory, 720)

	evidence, err := AbortGPUActivation(GPUActivationAbortOptions{
		DataDirectory:            directory,
		EvidenceDirectory:        evidenceDirectory,
		ExpectedActivationHeight: 720,
	})
	if err != nil {
		t.Fatalf("abort activation: %v", err)
	}
	if evidence.ChainTipHeight != 0 || evidence.ActivationHeight != 720 {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
	if _, err := os.Stat(filepath.Join(directory, gpuActivationRecordFilename)); !os.IsNotExist(err) {
		t.Fatalf("live activation record remains: %v", err)
	}
	preservedPath := filepath.Join(evidenceDirectory, gpuActivationRecordFilename)
	preserved, err := os.ReadFile(preservedPath)
	if err != nil {
		t.Fatalf("read preserved activation record: %v", err)
	}
	if string(preserved) != string(record) {
		t.Fatalf("preserved activation record changed: %q", preserved)
	}
	wantHash := sha256.Sum256(record)
	if evidence.ActivationRecordSHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("activation record hash = %q", evidence.ActivationRecordSHA256)
	}
	manifestPath := filepath.Join(evidenceDirectory, gpuActivationAbortManifestFilename)
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatalf("stat evidence manifest: %v", err)
	}
	if runtime.GOOS != "windows" && manifestInfo.Mode().Perm() != 0600 {
		t.Fatalf("evidence manifest mode = %o, want 600", manifestInfo.Mode().Perm())
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read evidence manifest: %v", err)
	}
	var decoded GPUActivationAbortEvidence
	if err := json.Unmarshal(manifestData, &decoded); err != nil {
		t.Fatalf("decode evidence manifest: %v", err)
	}
	if decoded != *evidence {
		t.Fatalf("manifest evidence = %+v, returned %+v", decoded, *evidence)
	}
}

func TestGPUActivationAbortRejectsUnsafeInputs(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, string, string)
		want    string
	}{
		{
			name: "missing activation record",
			prepare: func(t *testing.T, directory, _ string) {
				writeAbortChain(t, directory)
			},
			want: "activation record does not exist",
		},
		{
			name: "mismatched activation height",
			prepare: func(t *testing.T, directory, _ string) {
				writeAbortFixture(t, directory, 720)
			},
			want: "expected 721",
		},
		{
			name: "existing evidence destination",
			prepare: func(t *testing.T, directory, evidenceDirectory string) {
				writeAbortFixture(t, directory, 720)
				if err := os.Mkdir(evidenceDirectory, 0700); err != nil {
					t.Fatal(err)
				}
			},
			want: "evidence destination already exists",
		},
		{
			name: "activation boundary reached",
			prepare: func(t *testing.T, directory, _ string) {
				writeAbortFixture(t, directory, 0)
			},
			want: "cannot be aborted",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			evidenceDirectory := filepath.Join(directory, "abort-evidence")
			test.prepare(t, directory, evidenceDirectory)
			expected := uint64(720)
			if test.name == "mismatched activation height" {
				expected = 721
			}
			if test.name == "activation boundary reached" {
				expected = 0
			}
			_, err := AbortGPUActivation(GPUActivationAbortOptions{
				DataDirectory:            directory,
				EvidenceDirectory:        evidenceDirectory,
				ExpectedActivationHeight: expected,
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
			if _, statErr := os.Stat(filepath.Join(directory, gpuActivationRecordFilename)); test.name != "missing activation record" && statErr != nil {
				t.Fatalf("failed abort changed live record: %v", statErr)
			}
		})
	}
}

func TestGPUActivationAbortRejectsActiveNodeLock(t *testing.T) {
	directory := t.TempDir()
	writeAbortFixture(t, directory, 720)
	lock, err := LockDataDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()

	_, err = AbortGPUActivation(GPUActivationAbortOptions{
		DataDirectory:            directory,
		EvidenceDirectory:        filepath.Join(directory, "abort-evidence"),
		ExpectedActivationHeight: 720,
	})
	if err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("active-node abort error = %v", err)
	}
}

func TestGPUActivationAbortRejectsInsecureOrSymlinkedRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission and symlink contract")
	}
	for _, test := range []struct {
		name    string
		prepare func(*testing.T, string)
	}{
		{
			name: "insecure",
			prepare: func(t *testing.T, directory string) {
				writeAbortFixture(t, directory, 720)
				if err := os.Chmod(filepath.Join(directory, gpuActivationRecordFilename), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlinked",
			prepare: func(t *testing.T, directory string) {
				writeAbortChain(t, directory)
				target := filepath.Join(directory, "target.json")
				if err := os.WriteFile(target, []byte("{\"gpu_v1_activation_height\":720}\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(directory, gpuActivationRecordFilename)); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			test.prepare(t, directory)
			if _, err := AbortGPUActivation(GPUActivationAbortOptions{
				DataDirectory:            directory,
				EvidenceDirectory:        filepath.Join(directory, "abort-evidence"),
				ExpectedActivationHeight: 720,
			}); err == nil {
				t.Fatal("unsafe activation record accepted")
			}
		})
	}
}

func writeAbortFixture(t *testing.T, directory string, activationHeight uint64) []byte {
	t.Helper()
	writeAbortChain(t, directory)
	record := []byte("{\"gpu_v1_activation_height\":" + strconv.FormatUint(activationHeight, 10) + "}\n")
	if err := os.WriteFile(filepath.Join(directory, gpuActivationRecordFilename), record, 0600); err != nil {
		t.Fatalf("write activation record: %v", err)
	}
	return record
}

func writeAbortChain(t *testing.T, directory string) {
	t.Helper()
	if err := blockchain.NewChain().SaveToFile(filepath.Join(directory, gpuActivationChainFilename)); err != nil {
		t.Fatalf("write chain: %v", err)
	}
}
