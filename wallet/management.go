package wallet

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BackupEncrypted verifies the source wallet with password, copies the
// encrypted wallet file to a new path without overwriting an existing file,
// and verifies that the backup restores to the same address.
func BackupEncrypted(sourcePath, destinationPath, password string) (string, error) {
	if sourcePath == "" || destinationPath == "" {
		return "", fmt.Errorf("source and destination wallet paths are required")
	}
	if filepath.Clean(sourcePath) == filepath.Clean(destinationPath) {
		return "", fmt.Errorf("backup destination must differ from source wallet")
	}

	sourceWallet, err := LoadEncrypted(sourcePath, password)
	if err != nil {
		return "", fmt.Errorf("source wallet verification failed: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source wallet: %w", err)
	}
	defer source.Close()

	directory := filepath.Dir(destinationPath)
	if directory != "." {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return "", fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create backup wallet: %w", err)
	}
	keep := false
	defer func() {
		_ = destination.Close()
		if !keep {
			_ = os.Remove(destinationPath)
		}
	}()

	if _, err := io.Copy(destination, source); err != nil {
		return "", fmt.Errorf("failed to copy encrypted wallet: %w", err)
	}
	if err := destination.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync backup wallet: %w", err)
	}
	if err := destination.Close(); err != nil {
		return "", fmt.Errorf("failed to close backup wallet: %w", err)
	}

	backupWallet, err := LoadEncrypted(destinationPath, password)
	if err != nil {
		return "", fmt.Errorf("backup verification failed: %w", err)
	}
	if backupWallet.Address != sourceWallet.Address {
		return "", fmt.Errorf("backup address mismatch")
	}

	keep = true
	return sourceWallet.Address, nil
}

// ChangeEncryptedPassword verifies the current password and replaces the
// encrypted wallet with a freshly encrypted copy using the new password. The
// original file is first moved aside so replacement works consistently on
// platforms where rename cannot overwrite an existing destination. If the
// final replacement fails, the original is restored.
func ChangeEncryptedPassword(path, currentPassword, newPassword string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("wallet path cannot be empty")
	}
	if currentPassword == newPassword {
		return "", fmt.Errorf("new wallet password must differ from current password")
	}

	w, err := LoadEncrypted(path, currentPassword)
	if err != nil {
		return "", fmt.Errorf("current wallet password verification failed: %w", err)
	}
	if len(newPassword) < 12 {
		return "", fmt.Errorf("new wallet password must contain at least 12 characters")
	}

	directory := filepath.Dir(path)
	temp, err := os.CreateTemp(directory, ".sudharma-wallet-rotate-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary wallet file: %w", err)
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("failed to prepare temporary wallet file: %w", err)
	}
	defer os.Remove(tempPath)

	if err := w.SaveEncrypted(tempPath, newPassword); err != nil {
		return "", fmt.Errorf("failed to encrypt wallet with new password: %w", err)
	}
	if err := os.Chmod(tempPath, 0600); err != nil {
		return "", fmt.Errorf("failed to secure temporary wallet permissions: %w", err)
	}

	check, err := LoadEncrypted(tempPath, newPassword)
	if err != nil {
		return "", fmt.Errorf("new encrypted wallet verification failed: %w", err)
	}
	if check.Address != w.Address {
		return "", fmt.Errorf("password rotation changed wallet address")
	}

	originalBackup, err := os.CreateTemp(directory, ".sudharma-wallet-original-*")
	if err != nil {
		return "", fmt.Errorf("failed to reserve original wallet recovery path: %w", err)
	}
	originalBackupPath := originalBackup.Name()
	if err := originalBackup.Close(); err != nil {
		_ = os.Remove(originalBackupPath)
		return "", fmt.Errorf("failed to prepare original wallet recovery path: %w", err)
	}
	if err := os.Remove(originalBackupPath); err != nil {
		return "", fmt.Errorf("failed to prepare original wallet recovery path: %w", err)
	}

	if err := os.Rename(path, originalBackupPath); err != nil {
		return "", fmt.Errorf("failed to preserve original wallet before replacement: %w", err)
	}
	restoreOriginal := true
	defer func() {
		if restoreOriginal {
			_ = os.Rename(originalBackupPath, path)
		}
	}()

	if err := os.Rename(tempPath, path); err != nil {
		if restoreErr := os.Rename(originalBackupPath, path); restoreErr != nil {
			return "", fmt.Errorf("wallet replacement failed (%v) and original restore failed (%v); original remains at %s", err, restoreErr, originalBackupPath)
		}
		restoreOriginal = false
		return "", fmt.Errorf("failed to replace wallet after password rotation: %w", err)
	}

	finalCheck, err := LoadEncrypted(path, newPassword)
	if err != nil || finalCheck.Address != w.Address {
		_ = os.Remove(path)
		if restoreErr := os.Rename(originalBackupPath, path); restoreErr != nil {
			return "", fmt.Errorf("rotated wallet verification failed and original restore failed: %v", restoreErr)
		}
		restoreOriginal = false
		if err != nil {
			return "", fmt.Errorf("rotated wallet verification failed: %w", err)
		}
		return "", fmt.Errorf("rotated wallet address mismatch")
	}

	restoreOriginal = false
	if err := os.Remove(originalBackupPath); err != nil {
		return "", fmt.Errorf("password changed but failed to remove temporary original wallet copy at %s: %w", originalBackupPath, err)
	}
	return w.Address, nil
}
