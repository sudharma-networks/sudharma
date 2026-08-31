package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sudharma-networks/sudharma/testnet"
)

type output struct {
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	ProtocolNetworkID string `json:"protocol_network_id"`
	GenesisHash       string `json:"genesis_hash"`
	DefaultP2PPort    int    `json:"default_p2p_port"`
	DefaultRPCPort    int    `json:"default_rpc_port"`
	DefaultDataDir    string `json:"default_data_directory"`
}

func main() {
	value := output{
		Name:              testnet.Name,
		Slug:              testnet.Slug,
		ProtocolNetworkID: testnet.ProtocolNetworkID,
		GenesisHash:       testnet.GenesisHash(),
		DefaultP2PPort:    testnet.DefaultP2PPort,
		DefaultRPCPort:    testnet.DefaultRPCPort,
		DefaultDataDir:    testnet.DefaultDataDir,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
