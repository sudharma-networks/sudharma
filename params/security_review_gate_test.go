package params

import "testing"

func TestMainnetReadinessUsesEvidenceBasedSecurityReviewGate(t *testing.T) {
	var foundEvidenceGate bool
	for _, gate := range MainnetReadiness() {
		switch gate.Name {
		case "security-review-evidence":
			foundEvidenceGate = true
			if gate.Ready {
				t.Fatal("security review gate must remain closed until audit findings and public review are complete")
			}
		case "independent-security-audit":
			t.Fatal("mainnet readiness must not require a paid/independent audit as the only accepted security-review path")
		}
	}

	if !foundEvidenceGate {
		t.Fatal("security-review-evidence gate missing")
	}
}
