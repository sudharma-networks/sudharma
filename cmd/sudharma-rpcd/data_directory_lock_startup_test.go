package main

import (
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/operations"
)

func TestNodeStartupRejectsDataDirectoryHeldByAnotherProcess(t *testing.T) {
	directory := t.TempDir()
	lock, err := operations.LockDataDirectory(directory)
	if err != nil {
		t.Fatalf("hold data directory: %v", err)
	}
	defer lock.Close()

	cfg := operations.DefaultConfig()
	cfg.DataDirectory = directory
	err = runNode(cfg)
	if err == nil || !strings.Contains(err.Error(), "data directory is already in use") {
		t.Fatalf("startup with locked data directory error = %v", err)
	}
}
