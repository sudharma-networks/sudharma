package params

// Security-review evidence sub-gates. Each attestation must be flipped only with
// recorded evidence. MainnetSecurityReviewEvidenceComplete() stays false until
// every sub-gate is true.
const (
	// InternalSecurityAuditRemediationComplete records that IS-001 through IS-009
	// from the 2026-09-01 maintainer-controlled internal audit have their required
	// Critical/High/Medium remediations merged with regression coverage. Step 5
	// reconciled #101-#104 against current main on 2026-09-02.
	InternalSecurityAuditRemediationComplete = true

	// SecurityRegressionRaceAdversarialGatePassed records retained passing
	// evidence for the repository security regression/race/adversarial gate.
	// Current-main CI #1073 re-verified this on 2026-09-02. Any later frozen
	// mainnet candidate must rerun the gate before launch review.
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
			Detail: "2026-09-01 internal audit required Critical/High/Medium remediations merged; Step 5 reconciled #101-#104",
		},
		{
			Name:   "security-regression-race-adversarial",
			Ready:  SecurityRegressionRaceAdversarialGatePassed,
			Detail: "scripts/security-regression-gate.sh passed; current-main CI #1073 re-verified 2026-09-02",
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
