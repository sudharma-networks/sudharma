package params

// Security-review evidence sub-gates. Each attestation must be flipped only with
// recorded evidence. MainnetSecurityReviewEvidenceComplete() stays false until
// every sub-gate is true.
const (
	// InternalSecurityAuditRemediationComplete records that IS-001 through IS-009
	// from the 2026-09-01 internal audit have remediation merged on the candidate
	// branch with passing regression coverage.
	InternalSecurityAuditRemediationComplete = true

	// SecurityRegressionRaceAdversarialGatePassed records that
	// scripts/security-regression-gate.sh passed on the frozen candidate commit.
	SecurityRegressionRaceAdversarialGatePassed = true

	// PhysicalGPUMiningEvidenceComplete records retained RTX 2060 localhost
	// staging acceptance and physical AMD/non-NVIDIA OpenCL 4 GiB+ evidence (#24).
	PhysicalGPUMiningEvidenceComplete = false

	// PublicCommunitySecurityReviewComplete records completion of the documented
	// public/community review window with no unresolved Critical/High reports.
	PublicCommunitySecurityReviewComplete = false
)

// MainnetSecurityReviewEvidenceComplete reports whether the zero-budget security
// review evidence path is complete. This never substitutes for an independent
// third-party audit.
func MainnetSecurityReviewEvidenceComplete() bool {
	return InternalSecurityAuditRemediationComplete &&
		SecurityRegressionRaceAdversarialGatePassed &&
		PhysicalGPUMiningEvidenceComplete &&
		PublicCommunitySecurityReviewComplete
}

// SecurityReviewEvidenceGates returns human-readable sub-gate status for CLIs.
func SecurityReviewEvidenceGates() []ReadinessGate {
	return []ReadinessGate{
		{
			Name:   "internal-audit-remediation",
			Ready:  InternalSecurityAuditRemediationComplete,
			Detail: "2026-09-01 internal audit Critical/High/Medium remediations merged",
		},
		{
			Name:   "security-regression-race-adversarial",
			Ready:  SecurityRegressionRaceAdversarialGatePassed,
			Detail: "scripts/security-regression-gate.sh passed on frozen candidate",
		},
		{
			Name:   "physical-gpu-mining-evidence",
			Ready:  PhysicalGPUMiningEvidenceComplete,
			Detail: "RTX 2060 localhost staging + AMD/non-NVIDIA OpenCL 4 GiB+ evidence retained (#24)",
		},
		{
			Name:   "public-community-security-review",
			Ready:  PublicCommunitySecurityReviewComplete,
			Detail: "documented public/community review window completed",
		},
	}
}
