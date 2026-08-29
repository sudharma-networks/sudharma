package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const dataDirectoryLockFilename = ".sudharma.lock"

// DataDirectoryLock prevents two processes from mutating the same node data
// directory. File existence is not ownership; the operating-system lock is.
type DataDirectoryLock struct {
	file      *os.File
	closeOnce sync.Once
	closeErr  error
}

// LockDataDirectory acquires an exclusive, non-blocking process lock for a
// node data directory. The caller must retain and close the returned lock.
func LockDataDirectory(path string) (*DataDirectoryLock, error) {
	if path == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	lockPath := filepath.Join(path, dataDirectoryLockFilename)
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open data directory lock: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return nil, fmt.Errorf("secure data directory lock: %w", err)
	}
	if err := tryLockFile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("data directory is already in use: %w", err)
	}
	return &DataDirectoryLock{file: file}, nil
}

// Close releases the process lock. It is safe to call more than once.
func (l *DataDirectoryLock) Close() error {
	if l == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		if l.file == nil {
			return
		}
		if err := unlockFile(l.file); err != nil {
			l.closeErr = fmt.Errorf("release data directory lock: %w", err)
		}
		if err := l.file.Close(); err != nil && l.closeErr == nil {
			l.closeErr = fmt.Errorf("close data directory lock: %w", err)
		}
	})
	return l.closeErr
}
