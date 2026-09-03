package params

import (
	"fmt"
	"strings"
)

const (
	// ProductionMiningAlgorithm is the only proof-of-work algorithm Sudharma
	// accepts for public-testnet and mainnet mining. The consensus-visible
	// identifier is sudharma-gpupow-v1 (Khushi Algorithm).
	ProductionMiningAlgorithm = "sudharma-gpupow-v1"

	// ProductionMiningBrand is the human-facing name for GPU-PoW. Branding
	// does not change hashes, vectors, or block serialization.
	ProductionMiningBrand = "Khushi"

	// ProductionMiningBackend is the only supported mining hardware class.
	// Sudharma is GPU-mined only; CPU mining and ASIC mining are not supported
	// on public-testnet or mainnet.
	ProductionMiningBackend = "gpu-only"

	// GPU-PoW v1 activation remains deliberately unarmed on both networks.
	// Any finite activation height requires a later dedicated consensus change
	// after the physical GPU and public/community review gates are complete.
	GPUV1ActivationDisabled      uint64 = ^uint64(0)
	GPUV1TestnetActivationHeight uint64 = GPUV1ActivationDisabled
	GPUV1MainnetActivationHeight uint64 = GPUV1ActivationDisabled

	MiningNetworkPublicTestnet = "public-testnet"
	MiningNetworkMainnet       = "mainnet"

	PublicTestnetMiningRPC = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"

	// PublicTestnetSeed1RPC and PublicTestnetSeed2RPC are the loopback-adjacent
	// sudharma-rpcd endpoints that the public-testnet HTTPS proxy failovers
	// between. They match deployment/testnet/public-rpc/lambda/index.mjs.
	PublicTestnetSeed1RPC = "http://172.31.10.171:29100"
	PublicTestnetSeed2RPC = "http://172.31.32.195:29100"

	// MainnetMiningRPC is reserved for the authorized mainnet HTTPS proxy.
	// Connecting still requires MainnetMiningAuthorized.
	MainnetMiningRPC = "https://mainnet.rpc.sudharma.invalid"

	// MainnetSeed1RPC and MainnetSeed2RPC are placeholders until operators
	// publish the mainnet seed VPC topology in the activation PR.
	MainnetSeed1RPC = "http://REPLACE_WITH_SEED1_PRIVATE_IP:29100"
	MainnetSeed2RPC = "http://REPLACE_WITH_SEED2_PRIVATE_IP:29100"

	// MainnetMiningAuthorized keeps mainnet mining closed until an explicit
	// launch decision. The algorithm and backend rules below still apply:
	// when mainnet opens it will be GPU-only, not CPU, not ASIC.
	MainnetMiningAuthorized = false
)

// GPUOnlyMiningMessage is returned by nodes and miners when a CPU, ASIC, or
// otherwise unsupported mining path is requested, and when GPU-PoW work is
// not yet being issued by a node.
const GPUOnlyMiningMessage = "Sudharma is GPU-only (Khushi / sudharma-gpupow-v1). CPU and ASIC mining are not supported on public-testnet or mainnet."

func NormalizeMiningNetwork(network string) string {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "", MiningNetworkPublicTestnet, "testnet", "sudharma-testnet-1":
		return MiningNetworkPublicTestnet
	case MiningNetworkMainnet, "sudharma-mainnet-1":
		return MiningNetworkMainnet
	default:
		return strings.ToLower(strings.TrimSpace(network))
	}
}

func MiningRPCForNetwork(network string) (string, error) {
	endpoints, err := MiningRPCEndpointsForNetwork(network)
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", fmt.Errorf("no mining RPC endpoints configured for network %q", network)
	}
	return endpoints[0], nil
}

// PublicTestnetMiningSeedRPCs returns the ordered seed RPC endpoints used by the
// public-testnet HTTPS proxy. GPU miners on trusted networks may connect to
// these directly; public clients should use PublicTestnetMiningRPC instead.
func PublicTestnetMiningSeedRPCs() []string {
	return []string{PublicTestnetSeed1RPC, PublicTestnetSeed2RPC}
}

// MainnetMiningSeedRPCs returns the ordered mainnet seed RPC endpoints once
// operators replace the placeholder private IPs during launch preparation.
func MainnetMiningSeedRPCs() []string {
	return []string{MainnetSeed1RPC, MainnetSeed2RPC}
}

// MiningRPCEndpointsForNetwork returns the mining RPC endpoints for a network.
// Public-testnet uses the same seed-1 then seed-2 path as the wallet, explorer,
// and faucet public proxy. Internet clients should prefer the first HTTPS URL;
// operators on the seed VPC may reach the private seed RPC URLs directly.
func MiningRPCEndpointsForNetwork(network string) ([]string, error) {
	switch NormalizeMiningNetwork(network) {
	case MiningNetworkPublicTestnet:
		return append([]string{PublicTestnetMiningRPC}, PublicTestnetMiningSeedRPCs()...), nil
	case MiningNetworkMainnet:
		if !MainnetMiningAuthorized {
			return nil, fmt.Errorf("mainnet mining is not authorized; Sudharma mainnet remains GPU-only and closed until launch")
		}
		return append([]string{MainnetMiningRPC}, MainnetMiningSeedRPCs()...), nil
	default:
		return nil, fmt.Errorf("unsupported mining network %q (use public-testnet or mainnet)", network)
	}
}

func ValidateProductionMiningAlgorithm(algorithm string) error {
	got := strings.ToLower(strings.TrimSpace(algorithm))
	if got == ProductionMiningAlgorithm || got == strings.ToLower(ProductionMiningBrand) {
		return nil
	}
	return fmt.Errorf("%s (got %q)", GPUOnlyMiningMessage, algorithm)
}

func ValidateMiningBackend(backend string) error {
	got := strings.ToLower(strings.TrimSpace(backend))
	switch got {
	case "", ProductionMiningBackend, "gpu", "cuda", "opencl", "nvidia", "amd":
		return nil
	case "cpu", "asic", "sha256", "sha256d", "sudharma-cpu-v1", "cpu-v1":
		return fmt.Errorf("%s", GPUOnlyMiningMessage)
	default:
		return fmt.Errorf("%s (unsupported backend %q)", GPUOnlyMiningMessage, backend)
	}
}

func CPUMiningSupported() bool  { return false }
func ASICMiningSupported() bool { return false }
func GPUMiningSupported() bool  { return true }
