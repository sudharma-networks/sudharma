package params

// ReadinessGate is one mainnet-launch prerequisite.
type ReadinessGate struct {
	Name   string
	Ready  bool
	Detail string
}

// MainnetSecurityReviewEvidenceComplete is computed from explicit sub-gates in
// security_review_evidence.go. It remains false until internal audit remediation,
// security regression/race/adversarial tests, physical GPU evidence, and the
// public/community review window are all recorded complete.

// MainnetReadiness reports engineering freeze status. A true launch still
// requires the human gates below (security-review evidence, timestamp freeze,
// seed topology, launch decision). This function never authorizes mainnet by
// itself.
func MainnetReadiness() []ReadinessGate {
	return []ReadinessGate{
		{
			Name:   "tokenomics-schedule",
			Ready:  len(MainnetEmissionEpochs) == int(MainnetEpochCount),
			Detail: "40-epoch 51M mainnet subsidy table is encoded",
		},
		{
			Name:   "network-identity-isolated",
			Ready:  NetworkMainnet != NetworkPublicTestnet && NetworkMainnet != "",
			Detail: "mainnet P2P ID is distinct from sudharma-testnet-1",
		},
		{
			Name:   "launch-authorization",
			Ready:  MainnetLaunchAuthorized,
			Detail: "MainnetLaunchAuthorized remains false until the launch decision",
		},
		{
			Name:   "genesis-timestamp-freeze",
			Ready:  MainnetGenesisTimestamp != 0,
			Detail: "mainnet genesis unix timestamp is still unset (0)",
		},
		{
			Name:   "security-review-evidence",
			Ready:  MainnetSecurityReviewEvidenceComplete(),
			Detail: "requires internal audit remediation, security regression/race/adversarial gate, physical GPU evidence, and completed public review; no claim of independent third-party audit",
		},
		{
			Name:   "mainnet-seed-topology",
			Ready:  false,
			Detail: "mainnet seed addresses are not published or deployed",
		},
		{
			Name:   "mainnet-mining-authorization",
			Ready:  MainnetMiningAuthorized,
			Detail: "GPU mining stays closed until mainnet launch arms MainnetMiningAuthorized",
		},
	}
}

// MainnetLaunchReady is true only when every readiness gate is Ready.
func MainnetLaunchReady() bool {
	for _, gate := range MainnetReadiness() {
		if !gate.Ready {
			return false
		}
	}
	return true
}
