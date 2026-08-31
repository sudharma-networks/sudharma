package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
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
	device := fs.Int("device", 0, "GPU device index")
	probe := fs.Bool("probe", false, "validate address, GPU-only policy and RPC without hashing")
	once := fs.Bool("once", false, "fetch one GPU job, hash it, submit, then exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sudharma GPU Miner — Khushi Algorithm")
	fmt.Fprintln(out, params.GPUOnlyMiningMessage)
	fmt.Fprintln(out, "NVIDIA CUDA and AMD/OpenCL GPUs only. Not CPU. Not ASIC.")
	fmt.Fprintln(out, "This is not the demand miner. Demand miner is unchanged and can run in parallel.")
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
		Device:  *device,
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
			fmt.Fprintf(out, "Connected to %s. GPU miner work is not being issued yet: %v\n", cfg.Network, err)
			fmt.Fprintln(out, "Waiting is GPU-only. CPU and ASIC mining will not start.")
			return nil
		}
		fmt.Fprintf(out, "GPU miner work ready at height %d algorithm %s reward %s\n", work.Height, work.Algorithm, work.RewardAddress)
		return nil
	}

	var gpu gpuminer.Backend
	hasher, err := gpuminer.DetectGPUHasher(*hasherDir)
	if err != nil {
		fmt.Fprintln(out, "Khushi GPU hasher not found in this folder. Mining public-testnet candidate blocks to your wallet.")
		fmt.Fprintln(out, "Demand miner is a separate process and is not started from this app.")
	} else {
		fmt.Fprintf(out, "GPU hasher: %s\n", hasher)
		gpu = gpuminer.CommandBackend{Path: hasher, Device: cfg.Device}
	}
	fmt.Fprintln(out, "Starting GPU miner. Rewards go to the address above. This will not mine on CPU or ASIC products.")

	loop := &gpuminer.Loop{
		Client:  client,
		Address: cfg.Address,
		Backend: gpu,
		Once:    *once,
		Log: func(format string, args ...any) {
			fmt.Fprintf(out, format+"\n", args...)
		},
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if *once {
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	} else {
		ctx, cancel = signal.NotifyContext(ctx, os.Interrupt)
	}
	defer cancel()

	accepted, err := loop.Run(ctx)
	if err != nil {
		return err
	}
	if *once && accepted < 1 {
		return fmt.Errorf("connected to %s but no GPU share was accepted yet", cfg.Network)
	}
	fmt.Fprintf(out, "Stopped GPU mining after %d accepted share(s).\n", accepted)
	return nil
}
