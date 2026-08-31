package gpuminer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sudharma-networks/sudharma/params"
)

var hasherNames = []string{
	"khushi-miner-nvidia.exe",
	"khushi-miner-opencl.exe",
	"khushi-miner-nvidia",
	"khushi-miner-opencl",
	"sudharma-khushi-nvidia.exe",
	"sudharma-khushi-opencl.exe",
}

func DetectGPUHasher(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("locate GPU miner: %w", err)
		}
		dir = filepath.Dir(executable)
	}
	for _, name := range hasherNames {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		lower := strings.ToLower(name)
		backend := "cuda"
		if strings.Contains(lower, "opencl") {
			backend = "opencl"
		}
		if err := params.ValidateMiningBackend(backend); err != nil {
			return "", err
		}
		return path, nil
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return "", fmt.Errorf("%s Place khushi-miner-nvidia%s or khushi-miner-opencl%s in the same folder as this miner", params.GPUOnlyMiningMessage, suffix, suffix)
}
