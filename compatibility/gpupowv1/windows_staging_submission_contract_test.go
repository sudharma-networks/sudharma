package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsHardwareRunnerUsesStagingChallengeAPI(t *testing.T) {
	script, err := os.ReadFile("../../scripts/windows/test-khushi-miner.ps1")
	if err != nil {
		t.Fatal(err)
	}
	text := string(script)
	for _, want := range []string{
		"/v1/mining/staging/challenge",
		"/v1/mining/staging/submit",
		"--staging-search",
		"--header-prefix-hex",
		"--target-hex",
		"--height",
		"--cache-nodes",
		"staging-solution-nonce=",
		"network-submission=staging-accepted",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hardware staging submission contract missing %q", want)
		}
	}
	if strings.Contains(text, "& $MinerPath @mineArgs") {
		t.Fatal("staging submission must not invoke the gated live --mine path")
	}
}
