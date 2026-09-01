package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestMainnetReadinessJSONReportsUnauthorizedLaunch(t *testing.T) {
	payload := buildOutput()
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()

	var decoded struct {
		LaunchAuthorized bool `json:"launch_authorized"`
		LaunchReady      bool `json:"launch_ready"`
		MiningStackReady bool `json:"mining_stack_ready"`
		Gates            []struct {
			Name  string `json:"name"`
			Ready bool   `json:"ready"`
		} `json:"gates"`
		MiningGates []struct {
			Name  string `json:"name"`
			Ready bool   `json:"ready"`
		} `json:"mining_gates"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.LaunchAuthorized || decoded.LaunchReady {
		t.Fatalf("payload = %+v", decoded)
	}
	var launchGate, securityReviewGate, timestampGate bool
	for _, gate := range decoded.Gates {
		switch gate.Name {
		case "launch-authorization":
			launchGate = !gate.Ready
		case "security-review-evidence":
			securityReviewGate = !gate.Ready
		case "genesis-timestamp-freeze":
			timestampGate = !gate.Ready
		}
	}
	if !launchGate || !securityReviewGate || !timestampGate {
		t.Fatalf("gates = %+v", decoded.Gates)
	}
	if !decoded.MiningStackReady {
		t.Fatal("mining_stack_ready must be true for testnet engineering gates")
	}
	var mainnetMiningGate bool
	for _, gate := range decoded.MiningGates {
		if gate.Name == "mainnet-mining" {
			mainnetMiningGate = !gate.Ready
		}
	}
	if !mainnetMiningGate {
		t.Fatalf("mining_gates = %+v", decoded.MiningGates)
	}
	if !strings.Contains(string(out), "mainnet") {
		t.Fatalf("output = %s", out)
	}
	if params.MainnetLaunchReady() {
		t.Fatal("MainnetLaunchReady must stay false before human gates close")
	}
}
