package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sudharma-networks/sudharma/params"
)

type gateJSON struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Detail string `json:"detail"`
}

type output struct {
	LaunchAuthorized       bool       `json:"launch_authorized"`
	LaunchReady            bool       `json:"launch_ready"`
	Gates                  []gateJSON `json:"gates"`
	SecurityReviewGates    []gateJSON `json:"security_review_gates"`
	SecurityReviewComplete bool       `json:"security_review_complete"`
	MiningStackReady       bool       `json:"mining_stack_ready"`
	MiningGates            []gateJSON `json:"mining_gates"`
}

func gatesToJSON(gates []params.ReadinessGate) []gateJSON {
	out := make([]gateJSON, 0, len(gates))
	for _, gate := range gates {
		out = append(out, gateJSON{
			Name:   gate.Name,
			Ready:  gate.Ready,
			Detail: gate.Detail,
		})
	}
	return out
}

func miningGatesToJSON(gates []params.MiningReadinessGate) []gateJSON {
	out := make([]gateJSON, 0, len(gates))
	for _, gate := range gates {
		out = append(out, gateJSON{
			Name:   gate.Name,
			Ready:  gate.Ready,
			Detail: gate.Detail,
		})
	}
	return out
}

func buildOutput() output {
	return output{
		LaunchAuthorized:       params.MainnetLaunchAuthorized,
		LaunchReady:            params.MainnetLaunchReady(),
		Gates:                  gatesToJSON(params.MainnetReadiness()),
		SecurityReviewGates:    gatesToJSON(params.SecurityReviewEvidenceGates()),
		SecurityReviewComplete: params.MainnetSecurityReviewEvidenceComplete(),
		MiningStackReady:       params.MiningStackReady(),
		MiningGates:            miningGatesToJSON(params.MiningReadiness()),
	}
}

func main() {
	payload := buildOutput()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if payload.LaunchReady {
		os.Exit(2)
	}
}
