package params

import "testing"

func TestMiningReadinessReportsTestnetStackReady(t *testing.T) {
	if !MiningStackReady() {
		t.Fatal("expected testnet mining stack gates to be ready")
	}
	for _, gate := range MiningReadiness() {
		if gate.Name == "mainnet-mining" && gate.Ready {
			t.Fatal("mainnet mining must stay unauthorized")
		}
	}
}
