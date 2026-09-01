package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/pool"
	"github.com/sudharma-networks/sudharma/pool/stratum"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("sudharma-pool", flag.ContinueOnError)
	configPath := fs.String("config", "", "pool operator JSON config")
	network := fs.String("network", "public-testnet", "Sudharma mining network")
	rpcURL := fs.String("rpc", "", "Sudharma mining RPC URL")
	payoutAddress := fs.String("payout-address", "", "pool payout wallet (40 hex chars)")
	payoutScheme := fs.String("payout-scheme", "pplns", "solo, pps, pplns, or fpps")
	poolDifficulty := fs.Uint("pool-difficulty", uint(pool.DefaultPoolDifficulty), "share difficulty for pool workers")
	stratumListen := fs.String("stratum-listen", ":3333", "Stratum v1 listen address")
	probe := fs.Bool("probe", false, "validate config and fetch one mining job, then exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*configPath, *network, *rpcURL, *payoutAddress, *payoutScheme, uint32(*poolDifficulty), *stratumListen)
	if err != nil {
		return fmt.Errorf("pool config: %w", err)
	}

	engine, err := newEngine(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if *probe {
		return probePool(ctx, cfg, engine)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if _, err := engine.RefreshWork(rootCtx); err != nil {
		return fmt.Errorf("initial work refresh: %w", err)
	}

	go func() {
		ticker := time.NewTicker(cfg.WorkPollInterval())
		defer ticker.Stop()
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-ticker.C:
				if _, err := engine.RefreshWork(rootCtx); err != nil {
					log.Printf("work refresh failed: %v", err)
				}
			}
		}
	}()

	server := stratum.NewServer(engine, log.Printf)
	log.Printf("Sudharma pool starting scheme=%s listen=%s payout=%s", cfg.PayoutScheme, cfg.StratumListen, cfg.PayoutAddress)
	if err := server.ListenAndServe(rootCtx, cfg.StratumListen); err != nil && rootCtx.Err() == nil {
		return fmt.Errorf("stratum server: %w", err)
	}
	return nil
}

func newEngine(cfg pool.Config) (*pool.Engine, error) {
	minerCfg := gpuminer.Config{
		Address: cfg.PayoutAddress,
		Network: cfg.Network,
		RPCURL:  cfg.RPCURL,
		RPCURLs: cfg.RPCURLs,
	}
	resolved, err := gpuminer.Resolve(minerCfg)
	if err != nil {
		return nil, err
	}
	client, err := gpuminer.NewFailoverClient(resolved.RPCURLs, 20*time.Second)
	if err != nil {
		return nil, err
	}
	return pool.NewEngine(cfg, client)
}

func probePool(ctx context.Context, cfg pool.Config, engine *pool.Engine) error {
	job, err := engine.RefreshWork(ctx)
	if err != nil {
		return fmt.Errorf("mining work fetch failed: %w", err)
	}
	payload := map[string]any{
		"status":          "ready",
		"payout_scheme":   cfg.PayoutScheme,
		"stratum_listen":  cfg.StratumListen,
		"payout_address":  cfg.PayoutAddress,
		"pool_difficulty": cfg.PoolDifficulty,
		"job_id":          job.ID,
		"height":          job.Height,
		"rpc_endpoints":   len(cfg.RPCURLs),
	}
	if cfg.RPCURL != "" {
		payload["rpc_url"] = cfg.RPCURL
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func loadConfig(path, network, rpcURL, payoutAddress, payoutScheme string, poolDifficulty uint32, stratumListen string) (pool.Config, error) {
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return pool.Config{}, err
		}
		return pool.LoadConfig(raw)
	}
	cfg := pool.DefaultConfig()
	cfg.Network = network
	cfg.RPCURL = rpcURL
	cfg.PayoutAddress = payoutAddress
	cfg.PayoutScheme = pool.PayoutScheme(payoutScheme)
	cfg.PoolDifficulty = poolDifficulty
	cfg.StratumListen = stratumListen
	return pool.ResolveConfig(cfg)
}
