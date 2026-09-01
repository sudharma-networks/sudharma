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
	"github.com/sudharma-networks/sudharma/gpuminer/stratum"
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
	configPath := fs.String("config", "", "optional GPU miner JSON config (deployment/testnet/gpu-miner*.example.json)")
	network := fs.String("network", params.MiningNetworkPublicTestnet, "mining network: public-testnet or mainnet")
	rpcURL := fs.String("rpc", "", "optional mining RPC URL override for local testing")
	backend := fs.String("backend", params.ProductionMiningBackend, "GPU backend: gpu-only, cuda, or opencl")
	hasherDir := fs.String("hasher-dir", "", "folder that contains the NVIDIA or AMD GPU hasher")
	device := fs.Int("device", 0, "GPU device index")
	probe := fs.Bool("probe", false, "validate address and RPC without hashing")
	once := fs.Bool("once", false, "fetch one GPU job, hash it, submit, then exit")
	stratumURL := fs.String("stratum", "", "Stratum pool URL (stratum+tcp://host:port) for pool mining")
	workerName := fs.String("worker", "default", "pool worker name appended as wallet.worker")
	poolPassword := fs.String("password", "x", "Stratum pool password (most pools use x)")
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

	if poolCfg, poolMode, err := resolvePoolConfig(*configPath, *stratumURL, *workerName, *poolPassword, reward); err != nil {
		return err
	} else if poolMode {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		if *once {
			var onceCancel context.CancelFunc
			ctx, onceCancel = context.WithTimeout(ctx, 30*time.Second)
			defer onceCancel()
		}
		return runPoolMiner(ctx, out, poolCfg, *once, *hasherDir, *device)
	}

	resolved, err := resolveMinerConfig(*configPath, reward, *network, *rpcURL, *backend, *device)
	if err != nil {
		return err
	}

	client, err := gpuminer.NewFailoverClient(resolved.RPCURLs, 15*time.Second)
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

	fmt.Fprintf(out, "Connecting to Sudharma %s …\n", resolved.Network)
	status, statusErr := client.NetworkStatus(ctx)
	if statusErr == nil {
		if err := gpuminer.ValidateNetworkStatus(status, resolved.ExpectedNetwork); err != nil {
			return err
		}
		fmt.Fprintf(out, "Connected via %s. Network height %d.\n", client.Endpoint(), status.Height)
	} else {
		fmt.Fprintf(out, "Connecting via %s. Waiting for mining work …\n", client.Endpoint())
	}

	fmt.Fprintf(out, "Reward address: %s\n", resolved.Address)
	fmt.Fprintln(out, "")

	if *probe {
		work, err := client.GetWork(ctx, resolved.Address)
		if err != nil {
			fmt.Fprintf(out, "Mining work is not live yet: %v\n", err)
			fmt.Fprintln(out, "This miner keeps waiting. It will not switch to CPU or ASIC mining.")
			return nil
		}
		fmt.Fprintf(out, "Mining work ready at height %d.\n", work.Height)
		return nil
	}

	if err := gpuminer.SaveAddress(resolved.Address); err != nil {
		fmt.Fprintf(out, "Note: could not remember wallet address for next time: %v\n", err)
	}

	var gpu gpuminer.Backend
	hasher, err := gpuminer.DetectGPUHasher(*hasherDir)
	if err == nil {
		gpu = gpuminer.CommandBackend{Path: hasher, Device: resolved.Device}
	}

	fmt.Fprintln(out, "Mining started. Block rewards go to your wallet address.")
	fmt.Fprintln(out, "Press Ctrl+C to stop.")
	fmt.Fprintln(out, "")

	loop := &gpuminer.Loop{
		Client:  client,
		Address: resolved.Address,
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

func resolveMinerConfig(configPath, reward, network, rpcURL, backend string, device int) (gpuminer.Config, error) {
	if strings.TrimSpace(configPath) != "" {
		fileCfg, err := gpuminer.LoadFileConfig(configPath)
		if err != nil {
			return gpuminer.Config{}, err
		}
		cfg, err := fileCfg.ToConfig()
		if err != nil {
			return gpuminer.Config{}, err
		}
		if strings.TrimSpace(reward) != "" {
			address, err := gpuminer.NormalizeRewardAddress(reward)
			if err != nil {
				return gpuminer.Config{}, err
			}
			cfg.Address = address
		}
		if strings.TrimSpace(backend) != "" {
			cfg.Backend = backend
		}
		if device != 0 {
			cfg.Device = device
		}
		return gpuminer.Resolve(cfg)
	}
	return gpuminer.Resolve(gpuminer.Config{
		Address: reward,
		Network: network,
		RPCURL:  rpcURL,
		Backend: backend,
		Device:  device,
	})
}

func shortAddress(address string) string {
	address = strings.TrimSpace(address)
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "…" + address[len(address)-4:]
}

type poolMinerConfig struct {
	StratumURL string
	Login      string
	Password   string
	Network    string
	Address    string
}

func resolvePoolConfig(configPath, stratumURL, workerName, password, reward string) (poolMinerConfig, bool, error) {
	if strings.TrimSpace(configPath) != "" {
		poolCfg, err := gpuminer.LoadPoolFileConfig(configPath)
		if err == nil {
			address := poolCfg.RewardAddress
			if strings.TrimSpace(reward) != "" {
				address, err = gpuminer.NormalizeRewardAddress(reward)
				if err != nil {
					return poolMinerConfig{}, false, err
				}
			}
			worker := poolCfg.WorkerName
			if strings.TrimSpace(workerName) != "" && workerName != "default" {
				worker = workerName
			}
			login, err := stratum.WorkerLogin(address, worker)
			if err != nil {
				return poolMinerConfig{}, false, err
			}
			pass := poolCfg.Password
			if strings.TrimSpace(password) != "" {
				pass = password
			}
			return poolMinerConfig{
				StratumURL: poolCfg.StratumURL,
				Login:      login,
				Password:   pass,
				Network:    poolCfg.Network(),
				Address:    address,
			}, true, nil
		}
	}
	if strings.TrimSpace(stratumURL) == "" {
		return poolMinerConfig{}, false, nil
	}
	address, err := gpuminer.NormalizeRewardAddress(reward)
	if err != nil {
		return poolMinerConfig{}, false, err
	}
	login, err := stratum.WorkerLogin(address, workerName)
	if err != nil {
		return poolMinerConfig{}, false, err
	}
	return poolMinerConfig{
		StratumURL: stratumURL,
		Login:      login,
		Password:   password,
		Address:    address,
	}, true, nil
}

func runPoolMiner(ctx context.Context, out io.Writer, cfg poolMinerConfig, once bool, hasherDir string, device int) error {
	fmt.Fprintf(out, "Connecting to pool %s as %s …\n", cfg.StratumURL, cfg.Login)
	fmt.Fprintln(out, "Pool shares accumulate through the operator payout scheme (PPS/PPLNS/etc.).")
	fmt.Fprintln(out, "Press Ctrl+C to stop.")
	fmt.Fprintln(out, "")

	if err := gpuminer.SaveAddress(cfg.Address); err != nil {
		fmt.Fprintf(out, "Note: could not remember wallet address for next time: %v\n", err)
	}

	var miner stratum.ShareMiner = stratum.ReferenceShareMiner{}
	if hasher, err := gpuminer.DetectGPUHasher(hasherDir); err == nil {
		miner = stratum.NewShareMiner(gpuminer.CommandBackend{Path: hasher, Device: device})
		fmt.Fprintf(out, "Using GPU hasher %s for pool shares.\n", hasher)
	} else {
		fmt.Fprintln(out, "No Khushi GPU hasher found; using reference share search for pool jobs.")
	}

	loop := &stratum.Loop{
		PoolURL:  cfg.StratumURL,
		Login:    cfg.Login,
		Password: cfg.Password,
		Miner:    miner,
		Once:     once,
		Log: func(format string, args ...any) {
			fmt.Fprintf(out, format+"\n", args...)
		},
	}
	shares, blocks, err := loop.Run(ctx)
	if err != nil {
		return err
	}
	if once && shares+blocks < 1 {
		return fmt.Errorf("connected to pool but no share was accepted yet")
	}
	fmt.Fprintf(out, "Stopped after %d share(s) and %d block(s).\n", shares, blocks)
	return nil
}
