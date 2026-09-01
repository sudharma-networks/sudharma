package params

import "testing"

func TestProductionMiningIsGPUOnlyOnTestnetAndMainnet(t *testing.T) {
	if ProductionMiningAlgorithm != "sudharma-gpupow-v1" {
		t.Fatalf("algorithm = %q", ProductionMiningAlgorithm)
	}
	if ProductionMiningBackend != "gpu-only" {
		t.Fatalf("backend = %q", ProductionMiningBackend)
	}
	if !GPUMiningSupported() || CPUMiningSupported() || ASICMiningSupported() {
		t.Fatal("Sudharma production mining must be GPU-only")
	}
	if MainnetMiningAuthorized {
		t.Fatal("mainnet mining must stay unauthorized until launch")
	}

	for _, network := range []string{MiningNetworkPublicTestnet, MiningNetworkMainnet, "testnet", "sudharma-mainnet-1"} {
		normalized := NormalizeMiningNetwork(network)
		if normalized != MiningNetworkPublicTestnet && normalized != MiningNetworkMainnet {
			t.Fatalf("network %q normalized to %q", network, normalized)
		}
	}
}

func TestValidateProductionMiningAlgorithmRejectsCPUAndASIC(t *testing.T) {
	if err := ValidateProductionMiningAlgorithm(ProductionMiningAlgorithm); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProductionMiningAlgorithm("Khushi"); err != nil {
		t.Fatal(err)
	}
	for _, algorithm := range []string{"sudharma-cpu-v1", "sha256d", "sha256", "asic", "scrypt"} {
		err := ValidateProductionMiningAlgorithm(algorithm)
		if err == nil {
			t.Fatalf("accepted %q", algorithm)
		}
		if got := err.Error(); got != GPUOnlyMiningMessage+" (got \""+algorithm+"\")" {
			t.Fatalf("algorithm %q error = %q", algorithm, got)
		}
	}
}

func TestValidateMiningBackendRejectsCPUAndASIC(t *testing.T) {
	for _, backend := range []string{"", "gpu", "gpu-only", "cuda", "opencl", "nvidia", "amd"} {
		if err := ValidateMiningBackend(backend); err != nil {
			t.Fatalf("backend %q: %v", backend, err)
		}
	}
	for _, backend := range []string{"cpu", "asic", "sha256d", "sudharma-cpu-v1"} {
		err := ValidateMiningBackend(backend)
		if err == nil {
			t.Fatalf("accepted backend %q", backend)
		}
		if err.Error() != GPUOnlyMiningMessage {
			t.Fatalf("backend %q error = %q", backend, err)
		}
	}
}

func TestMiningRPCForNetwork(t *testing.T) {
	url, err := MiningRPCForNetwork(MiningNetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if url != PublicTestnetMiningRPC {
		t.Fatalf("testnet rpc = %q", url)
	}

	endpoints, err := MiningRPCEndpointsForNetwork(MiningNetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 3 {
		t.Fatalf("endpoints = %#v", endpoints)
	}
	if endpoints[0] != PublicTestnetMiningRPC {
		t.Fatalf("primary endpoint = %q", endpoints[0])
	}
	if endpoints[1] != PublicTestnetSeed1RPC || endpoints[2] != PublicTestnetSeed2RPC {
		t.Fatalf("seed endpoints = %#v", endpoints[1:])
	}

	_, err = MiningRPCForNetwork(MiningNetworkMainnet)
	if err == nil {
		t.Fatal("mainnet mining must be refused until authorized")
	}
}
