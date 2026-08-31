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

	NetworkPublicTestnet = "public-testnet"
	NetworkMainnet       = "mainnet"

	PublicTestnetMiningRPC = "https://ja6a03avlc.execute-api.ap-south-1.amazonaws.com"

	// MainnetMiningRPC is reserved for the authorized mainnet endpoint.
	// Connecting still requires MainnetMiningAuthorized.
	MainnetMiningRPC = "https://mainnet.rpc.sudharma.invalid"

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
	case "", NetworkPublicTestnet, "testnet", "sudharma-testnet-1":
		return NetworkPublicTestnet
	case NetworkMainnet, "sudharma-mainnet-1":
		return NetworkMainnet
	default:
		return strings.ToLower(strings.TrimSpace(network))
	}
}

func MiningRPCForNetwork(network string) (string, error) {
	switch NormalizeMiningNetwork(network) {
	case NetworkPublicTestnet:
		return PublicTestnetMiningRPC, nil
	case NetworkMainnet:
		if !MainnetMiningAuthorized {
			return "", fmt.Errorf("mainnet mining is not authorized; Sudharma mainnet remains GPU-only and closed until launch")
		}
		return MainnetMiningRPC, nil
	default:
		return "", fmt.Errorf("unsupported mining network %q (use public-testnet or mainnet)", network)
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
