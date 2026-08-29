package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCUDAVectorSelfTestContract(t *testing.T) {
	paths := []string{
		filepath.Join("..", "cuda", "gpupow_v1_search.cuh"),
		filepath.Join("..", "cuda", "khushi_miner.cu"),
	}
	var source strings.Builder
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read CUDA source %s: %v", path, err)
		}
		source.Write(data)
		source.WriteByte('\n')
	}
	text := source.String()

	required := []string{
		"__global__ void khushi_vector_kernel",
		"--vector-self-test",
		"2a7c15fc6c84a67d43ff7074ac5835aa433145f89d10d1d9e36a99fe22da4b2b",
		"vector-self-test=ok",
		"613684e3f3b42773073fb9c99e71f2933eed301d450866fe9a5a5c0530a769bd",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("CUDA vector self-test source missing required token %q", token)
		}
	}
}
