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

func TestSecurityReviewEvidenceSubGatesStayClosedByDefault(t *testing.T) {
	if MainnetSecurityReviewEvidenceComplete() {
		t.Fatal("security review evidence must remain incomplete until every sub-gate closes")
	}
	for _, gate := range SecurityReviewEvidenceGates() {
		switch gate.Name {
		case "internal-audit-remediation", "security-regression-race-adversarial":
			if !gate.Ready {
				t.Fatalf("engineering sub-gate %q should be ready on candidate branch", gate.Name)
			}
		default:
			if gate.Ready {
				t.Fatalf("sub-gate %q must remain closed until evidence is recorded", gate.Name)
			}
		}
	}
}
