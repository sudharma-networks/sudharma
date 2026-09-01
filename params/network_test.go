package params

import "testing"

func TestDefaultNetworkRemainsPublicTestnet(t *testing.T) {
	if DefaultNetwork != NetworkPublicTestnet {
		t.Fatalf("default network changed: got %q", DefaultNetwork)
	}
	if NetworkPublicTestnet != "sudharma-testnet-1" {
		t.Fatalf("public testnet identity changed: got %q", NetworkPublicTestnet)
	}
}

func TestMainnetNetworkIDIsIsolated(t *testing.T) {
	if NetworkMainnet != "sudharma-mainnet-1" {
		t.Fatalf("mainnet identity changed: got %q", NetworkMainnet)
	}
	if NetworkMainnet == NetworkPublicTestnet {
		t.Fatal("mainnet and testnet share a network ID")
	}
}

func TestParseNetworkRejectsMainnetWhileUnauthorized(t *testing.T) {
	if MainnetLaunchAuthorized {
		t.Fatal("mainnet launch became authorized without an explicit decision")
	}
	if _, err := ParseNetwork("mainnet"); err == nil {
		t.Fatal("expected mainnet parse to fail while unauthorized")
	}
	got, err := ParseNetwork("public-testnet")
	if err != nil {
		t.Fatal(err)
	}
	if got != NetworkPublicTestnet {
		t.Fatalf("got %q", got)
	}
}

func TestMonetaryPolicyForNetworks(t *testing.T) {
	got, err := MonetaryPolicyFor(NetworkMainnet)
	if err != nil {
		t.Fatal(err)
	}
	if got != MonetaryPolicyMainnet {
		t.Fatalf("mainnet policy: got %d", got)
	}
	got, err = MonetaryPolicyFor(NetworkPublicTestnet)
	if err != nil {
		t.Fatal(err)
	}
	if got != MonetaryPolicyPublicTestnet {
		t.Fatalf("testnet policy: got %d", got)
	}
}

func TestMainnetLaunchIsNotReady(t *testing.T) {
	if MainnetLaunchReady() {
		t.Fatal("mainnet reported launch-ready while human gates remain open")
	}
	var unauthorized, unfrozen, unaudited bool
	for _, gate := range MainnetReadiness() {
		switch gate.Name {
		case "launch-authorization":
			unauthorized = !gate.Ready
		case "genesis-timestamp-freeze":
			unfrozen = !gate.Ready
		case "independent-security-audit":
			unaudited = !gate.Ready
		}
	}
	if !unauthorized || !unfrozen || !unaudited {
		t.Fatal("required unready gates were marked ready")
	}
}
