package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sudharma-mainnet-genesis-preview <timestamp-unix> [<timestamp-unix>...]")
		os.Exit(2)
	}

	for _, raw := range os.Args[1:] {
		ts, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid timestamp %q: %v\n", raw, err)
			os.Exit(1)
		}
		genesis := blockchain.NewMainnetGenesisBlock()
		genesis.Timestamp = int64(ts)
		genesis.UpdateMerkleRoot()
		payload := map[string]any{
			"network":           params.NetworkMainnet,
			"timestamp_unix":    ts,
			"hash":              genesis.Hash(),
			"merkle_root":       genesis.MerkleRoot,
			"launch_authorized": params.MainnetLaunchAuthorized,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
