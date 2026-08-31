package gpuminer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const rewardAddressFile = "reward-address.txt"

// AddressStoreDir returns the folder used to remember the miner's wallet address.
func AddressStoreDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("SUDHARMA_MINER_DATA_DIR")); override != "" {
		return override, nil
	}
	if runtime.GOOS == "windows" {
		root := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if root == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not set")
		}
		return filepath.Join(root, "Sudharma", "gpu-miner"), nil
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		return filepath.Join(xdg, "sudharma", "gpu-miner"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "sudharma", "gpu-miner"), nil
}

func rewardAddressPath() (string, error) {
	dir, err := AddressStoreDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, rewardAddressFile), nil
}

// LoadSavedAddress returns the remembered wallet address, if any.
func LoadSavedAddress() (string, error) {
	path, err := rewardAddressPath()
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	address := strings.ToLower(strings.TrimSpace(string(raw)))
	if address == "" {
		return "", nil
	}
	if err := ValidateRewardAddress(address); err != nil {
		return "", nil
	}
	return address, nil
}

// SaveAddress remembers the wallet address for one-click restarts.
func SaveAddress(address string) error {
	address = strings.ToLower(strings.TrimSpace(address))
	if err := ValidateRewardAddress(address); err != nil {
		return err
	}
	path, err := rewardAddressPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(address+"\n"), 0o600)
}
