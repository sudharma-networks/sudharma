package operations

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStressConflictingConcurrentActivationPersistsExactlyOne(t *testing.T) {
	const iterations = 2000
	for i := 0; i < iterations; i++ {
		dir, err := os.MkdirTemp("", "gpu-activation-stress-")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "gpu-activation.json")
		start := make(chan struct{})
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for _, h := range []uint64{1720, 1800} {
			h := h
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs <- persistGPUActivation(path, GPUActivationPolicy{GPUV1ActivationHeight: h})
			}()
		}
		close(start)
		wg.Wait()
		close(errs)
		successes := 0
		for err := range errs {
			if err == nil {
				successes++
			}
		}
		if successes != 1 {
			_ = os.RemoveAll(dir)
			t.Fatalf("iteration %d: successes=%d, want exactly 1", i, successes)
		}
		if _, err := os.ReadFile(path); err != nil {
			_ = os.RemoveAll(dir)
			t.Fatalf("iteration %d: final record unreadable: %v", i, err)
		}
		matches, err := filepath.Glob(filepath.Join(dir, ".gpu-activation.json.tmp-*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			_ = os.RemoveAll(dir)
			t.Fatalf("iteration %d: temporary files remain: %v", i, matches)
		}
		_ = os.RemoveAll(dir)
	}
}
