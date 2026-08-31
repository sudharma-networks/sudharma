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
	configPath := flag.String("config", "", "pool operator JSON config")
	network := flag.String("network", "public-testnet", "Sudharma mining network")
	rpcURL := flag.String("rpc", "", "Sudharma mining RPC URL")
	payoutAddress := flag.String("payout-address", "", "pool payout wallet (40 hex chars)")
	payoutScheme := flag.String("payout-scheme", "pplns", "solo, pps, pplns, or fpps")
	poolDifficulty := flag.Uint("pool-difficulty", uint(pool.DefaultPoolDifficulty), "share difficulty for pool workers")
	stratumListen := flag.String("stratum-listen", ":3333", "Stratum v1 listen address")
	flag.Parse()

	cfg, err := loadConfig(*configPath, *network, *rpcURL, *payoutAddress, *payoutScheme, uint32(*poolDifficulty), *stratumListen)
	if err != nil {
		log.Fatalf("pool config: %v", err)
	}

	minerCfg := gpuminer.Config{
		Address: cfg.PayoutAddress,
		Network: cfg.Network,
		RPCURL:  cfg.RPCURL,
		RPCURLs: cfg.RPCURLs,
	}
	resolved, err := gpuminer.Resolve(minerCfg)
	if err != nil {
		log.Fatalf("mining client config: %v", err)
	}
	client, err := gpuminer.NewFailoverClient(resolved.RPCURLs, 20*time.Second)
	if err != nil {
		log.Fatalf("mining client: %v", err)
	}

	engine, err := pool.NewEngine(cfg, client)
	if err != nil {
		log.Fatalf("pool engine: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if _, err := engine.RefreshWork(ctx); err != nil {
		log.Fatalf("initial work refresh: %v", err)
	}

	go func() {
		ticker := time.NewTicker(cfg.WorkPollInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := engine.RefreshWork(ctx); err != nil {
					log.Printf("work refresh failed: %v", err)
				}
			}
		}
	}()

	server := stratum.NewServer(engine, log.Printf)
	log.Printf("Sudharma pool starting scheme=%s listen=%s payout=%s", cfg.PayoutScheme, cfg.StratumListen, cfg.PayoutAddress)
	if err := server.ListenAndServe(ctx, cfg.StratumListen); err != nil && ctx.Err() == nil {
		log.Fatalf("stratum server: %v", err)
	}
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

func printConfig(cfg pool.Config) {
	raw, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(raw))
}
