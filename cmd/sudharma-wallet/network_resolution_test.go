package main

import (
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/rpc"
)

func TestSigningNetworkFromStatusDefaultsToPublicTestnet(t *testing.T) {
	t.Setenv("SUDHARMA_NETWORK", "")
	network, err := signingNetworkFromStatus(&rpc.NodeStatus{NetworkID: string(params.NetworkPublicTestnet)})
	if err != nil {
		t.Fatal(err)
	}
	if network != params.NetworkPublicTestnet {
		t.Fatalf("network = %q, want %q", network, params.NetworkPublicTestnet)
	}
}

func TestSigningNetworkFromStatusRejectsMissingCanonicalIdentity(t *testing.T) {
	t.Setenv("SUDHARMA_NETWORK", "")
	_, err := signingNetworkFromStatus(&rpc.NodeStatus{Network: "sudharma"})
	if err == nil || !strings.Contains(err.Error(), "network_id") {
		t.Fatalf("expected missing network_id error, got %v", err)
	}
}

func TestSigningNetworkFromStatusRejectsUnauthorizedMainnet(t *testing.T) {
	t.Setenv("SUDHARMA_NETWORK", "")
	_, err := signingNetworkFromStatus(&rpc.NodeStatus{NetworkID: string(params.NetworkMainnet)})
	if err == nil || !strings.Contains(err.Error(), "mainnet launch is not authorized") {
		t.Fatalf("expected unauthorized mainnet rejection, got %v", err)
	}
}

func TestWalletExpectedNetworkRejectsUnauthorizedMainnet(t *testing.T) {
	t.Setenv("SUDHARMA_NETWORK", "mainnet")
	_, err := walletExpectedNetwork()
	if err == nil || !strings.Contains(err.Error(), "mainnet launch is not authorized") {
		t.Fatalf("expected fail-closed mainnet selection, got %v", err)
	}
}
