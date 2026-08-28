package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestStagingVerifierArtifactWorkflow(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/khushi-staging-verifier.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, token := range []string{
		"runs-on: ubuntu-24.04",
		"./cmd/sudharma-gpupow-staging",
		"sudharma-gpupow-staging",
		"SHA256SUMS.txt",
		"khushi-staging-verifier-linux-amd64",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("staging verifier artifact workflow missing %q", token)
		}
	}
}
