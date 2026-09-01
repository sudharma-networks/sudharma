package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/rpc"
)

// walletExpectedNetwork resolves the operator-selected wallet network. The
// public testnet remains the default until mainnet launch authorization is
// deliberately enabled in params.
func walletExpectedNetwork() (params.NetworkID, error) {
	raw := strings.TrimSpace(os.Getenv("SUDHARMA_NETWORK"))
	if raw == "" {
		raw = "public-testnet"
	}
	return params.ParseNetwork(raw)
}

// signingNetworkFromStatus validates the canonical network identity reported
// by the RPC node before any transaction is signed. The legacy `network` label
// is intentionally not accepted as a chain-domain separator.
func signingNetworkFromStatus(status *rpc.NodeStatus) (params.NetworkID, error) {
	if status == nil {
		return "", fmt.Errorf("RPC status is unavailable")
	}

	expected, err := walletExpectedNetwork()
	if err != nil {
		return "", fmt.Errorf("wallet network selection is invalid: %w", err)
	}

	raw := strings.TrimSpace(status.NetworkID)
	if raw == "" {
		return "", fmt.Errorf("RPC status is missing canonical network_id")
	}
	network, err := params.ParseNetwork(raw)
	if err != nil {
		return "", fmt.Errorf("RPC reported unsupported network_id: %w", err)
	}
	if network != expected {
		return "", fmt.Errorf("RPC network mismatch: expected %q, got %q", expected, network)
	}
	return network, nil
}
