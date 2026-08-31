package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

type genesisInfo struct {
	Network              string `json:"network"`
	LaunchAuthorized     bool   `json:"launch_authorized"`
	GenesisTimestamp     uint64 `json:"genesis_timestamp"`
	GenesisTimestampNote string `json:"genesis_timestamp_note"`
	MerkleRoot           string `json:"merkle_root"`
	Hash                 string `json:"hash"`
	Difficulty           uint32 `json:"difficulty"`
	Algorithm            string `json:"mining_algorithm"`
	MiningBackend        string `json:"mining_backend"`
	MiningAuthorized     bool   `json:"mining_authorized"`
}

func buildGenesisInfo() genesisInfo {
	genesis := blockchain.NewMainnetGenesisBlock()
	note := "timestamp is frozen for operator review"
	if params.MainnetGenesisTimestamp == 0 {
		note = "timestamp is unset (0) until an operator freeze replaces params.MainnetGenesisTimestamp"
	}
	return genesisInfo{
		Network:              string(params.NetworkMainnet),
		LaunchAuthorized:     params.MainnetLaunchAuthorized,
		GenesisTimestamp:     params.MainnetGenesisTimestamp,
		GenesisTimestampNote: note,
		MerkleRoot:           genesis.MerkleRoot,
		Hash:                 genesis.Hash(),
		Difficulty:           genesis.Difficulty,
		Algorithm:            params.ProductionMiningAlgorithm,
		MiningBackend:        params.ProductionMiningBackend,
		MiningAuthorized:     params.MainnetMiningAuthorized,
	}
}

func main() {
	payload := buildGenesisInfo()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
