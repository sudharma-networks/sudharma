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
	StackReady              bool       `json:"stack_ready"`
	MainnetMiningAuthorized bool       `json:"mainnet_mining_authorized"`
	Gates                   []gateJSON `json:"gates"`
}

func buildOutput() output {
	gates := params.MiningReadiness()
	payload := output{
		StackReady:              params.MiningStackReady(),
		MainnetMiningAuthorized: params.MainnetMiningAuthorized,
		Gates:                   make([]gateJSON, 0, len(gates)),
	}
	for _, gate := range gates {
		payload.Gates = append(payload.Gates, gateJSON{
			Name:   gate.Name,
			Ready:  gate.Ready,
			Detail: gate.Detail,
		})
	}
	return payload
}

func main() {
	payload := buildOutput()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !payload.StackReady {
		os.Exit(2)
	}
}
