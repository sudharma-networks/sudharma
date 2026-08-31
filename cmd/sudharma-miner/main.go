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
	auto := fs.Bool("auto", false, "use saved wallet address and start mining immediately")
	network := fs.String("network", params.NetworkPublicTestnet, "mining network: public-testnet or mainnet")
	rpcURL := fs.String("rpc", "", "optional mining RPC URL override for local testing")
	backend := fs.String("backend", params.ProductionMiningBackend, "GPU backend: gpu-only, cuda, or opencl")
	hasherDir := fs.String("hasher-dir", "", "folder that contains the NVIDIA or AMD GPU hasher")
	device := fs.Int("device", 0, "GPU device index")
	probe := fs.Bool("probe", false, "validate address and RPC without hashing")
	once := fs.Bool("once", false, "fetch one GPU job, hash it, submit, then exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Fprintln(out, "Sudharma GPU Miner")
	fmt.Fprintln(out, "Paste your wallet address once. The miner connects to Sudharma public-testnet automatically.")
	fmt.Fprintln(out, "")

	reward, err := resolveRewardAddress(*address, *auto, in, out)
	if err != nil {
		return err
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

	client, err := gpuminer.NewClient(cfg.RPCURL, 15*time.Second)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if *once || *probe {
		var probeCancel context.CancelFunc
		ctx, probeCancel = context.WithTimeout(ctx, 30*time.Second)
		defer probeCancel()
	}

	fmt.Fprintf(out, "Connecting to Sudharma %s …\n", cfg.Network)
	status, statusErr := client.NetworkStatus(ctx)
	if statusErr == nil {
		fmt.Fprintf(out, "Connected. Network height %d.\n", status.Height)
	} else {
		fmt.Fprintf(out, "Connected to %s. Waiting for mining work …\n", cfg.RPCURL)
	}

	fmt.Fprintf(out, "Reward address: %s\n", cfg.Address)
	fmt.Fprintln(out, "")

	if *probe {
		work, err := client.GetWork(ctx, cfg.Address)
		if err != nil {
			fmt.Fprintf(out, "Mining work is not live yet: %v\n", err)
			fmt.Fprintln(out, "This miner keeps waiting. It will not switch to CPU or ASIC mining.")
			return nil
		}
		fmt.Fprintf(out, "Mining work ready at height %d.\n", work.Height)
		return nil
	}

	if err := gpuminer.SaveAddress(cfg.Address); err != nil {
		fmt.Fprintf(out, "Note: could not remember wallet address for next time: %v\n", err)
	}

	var gpu gpuminer.Backend
	hasher, err := gpuminer.DetectGPUHasher(*hasherDir)
	if err == nil {
		gpu = gpuminer.CommandBackend{Path: hasher, Device: cfg.Device}
	}

	fmt.Fprintln(out, "Mining started. Block rewards go to your wallet address.")
	fmt.Fprintln(out, "Press Ctrl+C to stop.")
	fmt.Fprintln(out, "")

	loop := &gpuminer.Loop{
		Client:  client,
		Address: cfg.Address,
		Backend: gpu,
		Once:    *once,
		Log: func(format string, args ...any) {
			fmt.Fprintf(out, format+"\n", args...)
		},
	}

	accepted, err := loop.Run(ctx)
	if err != nil {
		return err
	}
	if *once && accepted < 1 {
		return fmt.Errorf("connected but no block was accepted yet")
	}
	if accepted > 0 {
		fmt.Fprintf(out, "Stopped after %d accepted block(s).\n", accepted)
	}
	return nil
}

func resolveRewardAddress(flagAddress string, auto bool, in io.Reader, out io.Writer) (string, error) {
	if strings.TrimSpace(flagAddress) != "" {
		return gpuminer.NormalizeRewardAddress(flagAddress)
	}
	if auto {
		saved, err := gpuminer.LoadSavedAddress()
		if err != nil {
			return "", err
		}
		if saved == "" {
			return "", fmt.Errorf("no saved wallet address yet; run once and enter your address")
		}
		fmt.Fprintf(out, "Using saved wallet address %s\n", shortAddress(saved))
		return saved, nil
	}

	saved, err := gpuminer.LoadSavedAddress()
	if err != nil {
		return "", err
	}
	if saved != "" {
		fmt.Fprintf(out, "Saved wallet: %s\n", shortAddress(saved))
		fmt.Fprint(out, "Press Enter to start mining, or paste a different address: ")
		scanner := bufio.NewScanner(in)
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				return gpuminer.NormalizeRewardAddress(line)
			}
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return saved, nil
	}

	fmt.Fprint(out, "Wallet address (40 hex characters): ")
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return "", fmt.Errorf("wallet address is required")
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return gpuminer.NormalizeRewardAddress(scanner.Text())
}

func shortAddress(address string) string {
	address = strings.TrimSpace(address)
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "…" + address[len(address)-4:]
}
