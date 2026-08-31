package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/params"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader, out, errOut io.Writer) error {
	fs := flag.NewFlagSet("sudharma-miner", flag.ContinueOnError)
	fs.SetOutput(errOut)
	address := fs.String("address", "", "Sudharma wallet address that receives mining rewards")
	network := fs.String("network", params.NetworkPublicTestnet, "mining network: public-testnet or mainnet")
	rpcURL := fs.String("rpc", "", "optional mining RPC URL override for local testing")
	backend := fs.String("backend", params.ProductionMiningBackend, "GPU backend: gpu-only, cuda, or opencl")
	hasherDir := fs.String("hasher-dir", "", "folder that contains the NVIDIA or AMD GPU hasher")
	probe := fs.Bool("probe", false, "validate address, GPU-only policy and RPC without hashing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sudharma GPU Miner — Khushi Algorithm")
	fmt.Fprintln(out, params.GPUOnlyMiningMessage)
	fmt.Fprintln(out, "NVIDIA CUDA and AMD/OpenCL GPUs only. Not CPU. Not ASIC.")
	fmt.Fprintln(out, "")

	reward := strings.TrimSpace(*address)
	if reward == "" {
		fmt.Fprint(out, "Paste your 40-character Sudharma wallet address: ")
		scanner := bufio.NewScanner(in)
		if scanner.Scan() {
			reward = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	cfg, err := gpuminer.Resolve(gpuminer.Config{
		Address: reward,
		Network: *network,
		RPCURL:  *rpcURL,
		Backend: *backend,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Network: %s\n", cfg.Network)
	fmt.Fprintf(out, "RPC: %s\n", cfg.RPCURL)
	fmt.Fprintf(out, "Reward address: %s\n", cfg.Address)
	fmt.Fprintf(out, "Algorithm: %s (%s)\n", params.ProductionMiningAlgorithm, params.ProductionMiningBrand)

	client, err := gpuminer.NewClient(cfg.RPCURL, 15*time.Second)
	if err != nil {
		return err
	}

	if *probe {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		work, err := client.GetWork(ctx, cfg.Address)
		if err != nil {
			fmt.Fprintf(out, "Connected to %s. GPU-PoW work is not being issued yet: %v\n", cfg.Network, err)
			return nil
		}
		fmt.Fprintf(out, "GPU work ready at height %d algorithm %s\n", work.Height, work.Algorithm)
		return nil
	}

	hasher, err := gpuminer.DetectGPUHasher(*hasherDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "GPU hasher: %s\n", hasher)
	fmt.Fprintln(out, "Starting GPU mining. CPU and ASIC paths are disabled.")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	work, err := client.GetWork(ctx, cfg.Address)
	if err != nil {
		return fmt.Errorf("connected to %s but GPU-PoW work is not active yet: %w", cfg.Network, err)
	}
	fmt.Fprintf(out, "Received GPU work at height %d. Keep this window open while the GPU hasher searches.\n", work.Height)
	return nil
}
