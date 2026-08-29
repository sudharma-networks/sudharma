package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestAbortCLIRequiresExplicitInputsAndConfirmation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "data directory", args: nil, want: "data-dir is required"},
		{name: "evidence directory", args: []string{"-data-dir", "node"}, want: "evidence-dir is required"},
		{
			name: "activation height",
			args: []string{"-data-dir", "node", "-evidence-dir", "evidence"},
			want: "expected-activation-height is required",
		},
		{
			name: "confirmation",
			args: []string{
				"-data-dir", "node",
				"-evidence-dir", "evidence",
				"-expected-activation-height", "720",
			},
			want: "confirm-abort is required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			err := runAbortCLI(test.args, &output)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestAbortCLIRejectsMalformedActivationHeight(t *testing.T) {
	err := runAbortCLI([]string{
		"-data-dir", "node",
		"-evidence-dir", "evidence",
		"-expected-activation-height", "not-a-height",
		"-confirm-abort",
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("malformed height error = %v", err)
	}
}

func TestAbortCLIExecutesConfirmedOfflineAbort(t *testing.T) {
	directory := t.TempDir()
	evidenceDirectory := filepath.Join(directory, "abort-evidence")
	if err := blockchain.NewChain().SaveToFile(filepath.Join(directory, "sudharma-chain.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "sudharma-gpu-v1-activation.json"),
		[]byte("{\"gpu_v1_activation_height\":720}\n"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	err := runAbortCLI([]string{
		"-data-dir", directory,
		"-evidence-dir", evidenceDirectory,
		"-expected-activation-height", "720",
		"-confirm-abort",
	}, &output)
	if err != nil {
		t.Fatalf("confirmed abort: %v", err)
	}
	if !strings.Contains(output.String(), "activation_abort=completed") ||
		!strings.Contains(output.String(), "activation_height=720") {
		t.Fatalf("unexpected output: %q", output.String())
	}
	if _, err := os.Stat(filepath.Join(evidenceDirectory, "sudharma-gpu-v1-activation.json")); err != nil {
		t.Fatalf("preserved record missing: %v", err)
	}
}
