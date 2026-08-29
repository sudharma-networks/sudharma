package operations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirectoryLockRejectsConcurrentOwnerAndReleases(t *testing.T) {
	directory := t.TempDir()
	first, err := LockDataDirectory(directory)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}

	if _, err := LockDataDirectory(directory); err == nil {
		t.Fatal("second owner acquired the same data directory")
	}
	if err := first.Close(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}

	second, err := LockDataDirectory(directory)
	if err != nil {
		t.Fatalf("reacquire released lock: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("release second lock: %v", err)
	}
}

func TestDataDirectoryLockIgnoresUnlockedStaleFile(t *testing.T) {
	directory := t.TempDir()
	lockPath := filepath.Join(directory, dataDirectoryLockFilename)
	if err := os.WriteFile(lockPath, []byte("stale\n"), 0600); err != nil {
		t.Fatalf("write stale lock file: %v", err)
	}

	lock, err := LockDataDirectory(directory)
	if err != nil {
		t.Fatalf("stale lock file blocked acquisition: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("release lock: %v", err)
	}
}

func TestDataDirectoryLocksAreScopedPerDirectory(t *testing.T) {
	first, err := LockDataDirectory(t.TempDir())
	if err != nil {
		t.Fatalf("acquire first directory: %v", err)
	}
	defer first.Close()

	second, err := LockDataDirectory(t.TempDir())
	if err != nil {
		t.Fatalf("unrelated directory lock conflicted: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("release second directory: %v", err)
	}
}

func TestDataDirectoryLockRejectsEmptyPath(t *testing.T) {
	if _, err := LockDataDirectory(""); err == nil {
		t.Fatal("empty data directory accepted")
	}
}
