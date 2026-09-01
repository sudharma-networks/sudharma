package params

// MiningReadinessGate reports one public mining capability gate.
type MiningReadinessGate struct {
	Name   string
	Ready  bool
	Detail string
}

// MiningReadiness summarizes the encoded GPU mining stack. Testnet solo and
// pool paths are engineering-ready; mainnet mining remains gated separately.
func MiningReadiness() []MiningReadinessGate {
	return []MiningReadinessGate{
		{
			Name:   "solo-http-api",
			Ready:  true,
			Detail: "POST /v1/mining/work and /v1/mining/submit for GPU candidate blocks",
		},
		{
			Name:   "pool-stratum-stack",
			Ready:  true,
			Detail: "reference sudharma-pool operator with Stratum v1 and gpuminer worker client",
		},
		{
			Name:   "payout-schemes",
			Ready:  true,
			Detail: "PPS, PPLNS, SOLO, and FPPS payout ledgers encoded in pool/",
		},
		{
			Name:   "mainnet-mining",
			Ready:  MainnetMiningAuthorized,
			Detail: "mainnet GPU mining stays closed until MainnetMiningAuthorized",
		},
	}
}

// MiningStackReady is true when every non-mainnet mining gate is ready.
func MiningStackReady() bool {
	for _, gate := range MiningReadiness() {
		if gate.Name == "mainnet-mining" {
			continue
		}
		if !gate.Ready {
			return false
		}
	}
	return true
}
