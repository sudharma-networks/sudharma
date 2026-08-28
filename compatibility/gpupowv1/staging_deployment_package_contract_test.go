package gpupowv1

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStagingVerifierArtifactIncludesDeploymentBundle(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "khushi-staging-verifier.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read staging verifier workflow: %v", err)
	}
	workflowText := string(workflow)
	for _, token := range []string{
		"deployment/staging/sudharma-gpupow-staging.service",
		"deployment/staging/nginx-staging.example.conf",
		"deployment/staging/install-staging-verifier.sh",
		"dist/deployment",
	} {
		if !strings.Contains(workflowText, token) {
			t.Fatalf("staging verifier workflow missing deployment bundle token %q", token)
		}
	}

	installerPath := filepath.Join("..", "..", "deployment", "staging", "install-staging-verifier.sh")
	installer, err := os.ReadFile(installerPath)
	if err != nil {
		t.Fatalf("read staging verifier installer: %v", err)
	}
	installerText := string(installer)
	for _, token := range []string{
		"SHA256SUMS.txt",
		"sha256sum -c",
		"/usr/local/bin/sudharma-gpupow-staging",
		"sudharma-staging",
		"sudharma-gpupow-staging.service",
		"systemctl daemon-reload",
		"systemctl enable --now sudharma-gpupow-staging.service",
		"127.0.0.1:28646",
	} {
		if !strings.Contains(installerText, token) {
			t.Fatalf("staging verifier installer missing %q", token)
		}
	}
	if strings.Contains(installerText, "sudharma.service") {
		t.Fatal("staging verifier installer must not modify the live Sudharma node service")
	}
}
