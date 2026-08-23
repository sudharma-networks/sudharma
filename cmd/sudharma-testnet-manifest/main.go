package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sudharma-networks/sudharma/testnet"
)

func main() {
	profilePath := flag.String("profile", "", "path to public testnet profile JSON")
	flag.Parse()
	if *profilePath == "" {
		fmt.Fprintln(os.Stderr, "sudharma-testnet-manifest: -profile is required")
		os.Exit(2)
	}
	profile, err := testnet.LoadProfile(*profilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sudharma-testnet-manifest: %v\n", err)
		os.Exit(1)
	}
	manifest, err := testnet.NewLaunchManifest(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sudharma-testnet-manifest: launch not ready: %v\n", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		fmt.Fprintf(os.Stderr, "sudharma-testnet-manifest: encode: %v\n", err)
		os.Exit(1)
	}
}
