package wallet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupEncryptedVerifiesSameWallet(t *testing.T) {
	original, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	password := "correct-horse-battery-staple"
	dir := t.TempDir()
	source := filepath.Join(dir, "wallet.json")
	backup := filepath.Join(dir, "backup", "wallet-backup.json")
	if err := original.SaveEncrypted(source, password); err != nil {
		t.Fatal(err)
	}

	address, err := BackupEncrypted(source, backup, password)
	if err != nil {
		t.Fatal(err)
	}
	if address != original.Address {
		t.Fatalf("backup returned wrong address: got %s want %s", address, original.Address)
	}
	restored, err := LoadEncrypted(backup, password)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Address != original.Address {
		t.Fatalf("backup restored wrong address: got %s want %s", restored.Address, original.Address)
	}
	info, err := os.Stat(backup)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("backup wallet permissions too broad: %o", info.Mode().Perm())
	}
}

func TestBackupEncryptedRejectsBadPasswordAndOverwrite(t *testing.T) {
	original, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "wallet.json")
	backup := filepath.Join(dir, "backup.json")
	if err := original.SaveEncrypted(source, "this-is-the-correct-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := BackupEncrypted(source, backup, "this-is-the-wrong-password"); err == nil {
		t.Fatal("backup accepted incorrect password")
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatal("failed backup left a destination file behind")
	}
	if _, err := BackupEncrypted(source, backup, "this-is-the-correct-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := BackupEncrypted(source, backup, "this-is-the-correct-password"); err == nil {
		t.Fatal("backup overwrote an existing backup file")
	}
}

func TestChangeEncryptedPasswordPreservesWallet(t *testing.T) {
	original, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "wallet.json")
	oldPassword := "this-is-the-old-password"
	newPassword := "this-is-the-new-password"
	if err := original.SaveEncrypted(path, oldPassword); err != nil {
		t.Fatal(err)
	}

	address, err := ChangeEncryptedPassword(path, oldPassword, newPassword)
	if err != nil {
		t.Fatal(err)
	}
	if address != original.Address {
		t.Fatalf("rotation returned wrong address: got %s want %s", address, original.Address)
	}
	if _, err := LoadEncrypted(path, oldPassword); err == nil {
		t.Fatal("old password still decrypts rotated wallet")
	}
	rotated, err := LoadEncrypted(path, newPassword)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Address != original.Address {
		t.Fatalf("rotation changed wallet address: got %s want %s", rotated.Address, original.Address)
	}
}

func TestChangeEncryptedPasswordFailureKeepsOriginal(t *testing.T) {
	original, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "wallet.json")
	password := "this-is-the-correct-password"
	if err := original.SaveEncrypted(path, password); err != nil {
		t.Fatal(err)
	}

	if _, err := ChangeEncryptedPassword(path, "wrong-current-password", "a-valid-new-password"); err == nil {
		t.Fatal("password rotation accepted wrong current password")
	}
	stillOriginal, err := LoadEncrypted(path, password)
	if err != nil {
		t.Fatalf("original wallet was damaged after failed rotation: %v", err)
	}
	if stillOriginal.Address != original.Address {
		t.Fatal("original wallet address changed after failed rotation")
	}
}
