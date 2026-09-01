package gpuminer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadAddress(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUDHARMA_MINER_DATA_DIR", dir)

	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := SaveAddress(address); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSavedAddress()
	if err != nil {
		t.Fatal(err)
	}
	if got != address {
		t.Fatalf("got %q", got)
	}
	path := filepath.Join(dir, rewardAddressFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("address file permissions too open: %o", info.Mode().Perm())
	}
}

func TestLoadSavedAddressMissingIsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUDHARMA_MINER_DATA_DIR", dir)
	got, err := LoadSavedAddress()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
}
