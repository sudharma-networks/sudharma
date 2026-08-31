package gpuminer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sudharma-networks/sudharma/params"
)

const defaultGPUCacheNodes uint32 = 8

type CommandBackend struct {
	Path   string
	Device int
	Run    func(ctx context.Context, path string, args []string) ([]byte, error)
}

func (c CommandBackend) Name() string {
	lower := strings.ToLower(c.Path)
	if strings.Contains(lower, "opencl") {
		return "opencl"
	}
	if strings.Contains(lower, "nvidia") || strings.Contains(lower, "cuda") {
		return "cuda"
	}
	return params.ProductionMiningBackend
}

func (c CommandBackend) Search(ctx context.Context, work Work) (uint64, error) {
	if err := params.ValidateMiningBackend(c.Name()); err != nil {
		return 0, err
	}
	if strings.TrimSpace(c.Path) == "" {
		return 0, fmt.Errorf("%s GPU hasher is missing", params.GPUOnlyMiningMessage)
	}
	args := HasherArgs(work, c.Device)
	run := c.Run
	if run == nil {
		run = defaultHasherRun
	}
	out, err := run(ctx, c.Path, args)
	if err != nil {
		return 0, fmt.Errorf("GPU hasher failed: %w", err)
	}
	nonce, err := ParseHasherNonce(out)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func HasherArgs(work Work, device int) []string {
	cacheNodes := work.CacheNodes
	if cacheNodes == 0 {
		cacheNodes = defaultGPUCacheNodes
	}
	args := []string{
		"--staging-search",
		"--header-prefix-hex", strings.TrimSpace(work.HeaderPrefix),
		"--target-hex", strings.TrimSpace(work.Target),
		"--height", strconv.FormatUint(work.Height, 10),
		"--cache-nodes", strconv.FormatUint(uint64(cacheNodes), 10),
	}
	if device > 0 {
		args = append([]string{"--device", strconv.Itoa(device)}, args...)
	}
	return args
}

func ParseHasherNonce(output []byte) (uint64, error) {
	for _, line := range bytes.Split(output, []byte("\n")) {
		text := strings.TrimSpace(string(line))
		if value, ok := strings.CutPrefix(text, "staging-solution-nonce="); ok {
			nonce, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid GPU hasher nonce: %w", err)
			}
			return nonce, nil
		}
	}
	return 0, fmt.Errorf("GPU hasher did not print staging-solution-nonce; CPU fallback is not allowed")
}

func defaultHasherRun(ctx context.Context, path string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = filepath.Dir(path)
	return cmd.CombinedOutput()
}
