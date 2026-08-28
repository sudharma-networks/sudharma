package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestPublicTestMiningDocsMatchHardwareEvidenceRunner(t *testing.T) {
	raw, err := os.ReadFile("../../docs/test-mining/PUBLIC_TEST_MINING.md")
	if err != nil {
		t.Fatalf("read public test-mining guide: %v", err)
	}
	text := string(raw)

	for _, token := range []string{
		"-EvidenceDirectory",
		"windows_version=",
		"video_driver_version=",
		"hardware-production-memory=passed",
		"hardware-production-vectors=passed",
		"hardware-vector-memory-and-benchmark=passed",
		"network-submission=not-requested",
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("public test-mining guide missing current evidence token %q", token)
		}
	}

	if strings.Contains(text, "-AllowMining") {
		t.Fatal("public test-mining guide documents nonexistent -AllowMining switch")
	}
}
