package params

import "fmt"

type MonetaryPolicy uint8

const (
	MonetaryPolicyPublicTestnet MonetaryPolicy = 1
	MonetaryPolicyMainnet       MonetaryPolicy = 2

	MainnetMaxSupplySUDH      uint64 = 51_000_000
	MainnetMaxSupply                 = MainnetMaxSupplySUDH * CoinDecimals
	MainnetFinalSubsidyHeight uint64 = 5_259_600
	MainnetEpochLength        uint64 = 131_490
	MainnetEpochCount         uint64 = 40
)

func ValidateMonetaryPolicy(policy MonetaryPolicy) error {
	switch policy {
	case MonetaryPolicyPublicTestnet, MonetaryPolicyMainnet:
		return nil
	default:
		return fmt.Errorf("unknown monetary policy %d", policy)
	}
}

func MaxSupplyFor(policy MonetaryPolicy) uint64 {
	switch policy {
	case MonetaryPolicyMainnet:
		return MainnetMaxSupply
	case MonetaryPolicyPublicTestnet:
		return MaxSupply
	default:
		return 0
	}
}
