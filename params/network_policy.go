package params

import "fmt"

// NetworkForMonetaryPolicy maps a monetary policy to its network identity.
func NetworkForMonetaryPolicy(policy MonetaryPolicy) (NetworkID, error) {
	switch policy {
	case MonetaryPolicyPublicTestnet:
		return NetworkPublicTestnet, nil
	case MonetaryPolicyMainnet:
		return NetworkMainnet, nil
	default:
		return "", fmt.Errorf("unknown monetary policy %d", policy)
	}
}
