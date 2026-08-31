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
	LaunchAuthorized bool       `json:"launch_authorized"`
	LaunchReady      bool       `json:"launch_ready"`
	Gates            []gateJSON `json:"gates"`
}

func buildOutput() output {
	gates := params.MainnetReadiness()
	payload := output{
		LaunchAuthorized: params.MainnetLaunchAuthorized,
		LaunchReady:      params.MainnetLaunchReady(),
		Gates:            make([]gateJSON, 0, len(gates)),
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
	if payload.LaunchReady {
		os.Exit(2)
	}
}
