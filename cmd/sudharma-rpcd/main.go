package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/operations"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/rpc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "sudharma-rpcd: %v\n", err)
		os.Exit(1)
	}
}
func run() error {
	configPath := flag.String("config", "", "optional production node JSON config")
	nodeID := flag.String("nodeid", "", "override node ID")
	p2pAddress := flag.String("listen", "", "override P2P listen address")
	rpcAddress := flag.String("rpc", "", "override HTTP RPC listen address")
	peerAddress := flag.String("peer", "", "optional additional peer address")
	dataDirectory := flag.String("datadir", "", "override node data directory")
	logJSON := flag.Bool("log-json", false, "emit structured JSON logs")
	flag.Parse()
	cfg, err := operations.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if *nodeID != "" {
		cfg.NodeID = *nodeID
	}
	if *p2pAddress != "" {
		cfg.P2PAddress = *p2pAddress
	}
	if *rpcAddress != "" {
		cfg.RPCAddress = *rpcAddress
	}
	if *dataDirectory != "" {
		cfg.DataDirectory = *dataDirectory
	}
	if *peerAddress != "" {
		cfg.Peers = append(cfg.Peers, *peerAddress)
	}
	if *logJSON {
		cfg.LogJSON = true
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	return runNode(cfg)
}

func runNode(cfg operations.Config) error {
	dataDirectoryLock, err := operations.LockDataDirectory(cfg.DataDirectory)
	if err != nil {
		return fmt.Errorf("lock node data directory: %w", err)
	}
	defer dataDirectoryLock.Close()

	log := operations.NewLogger(os.Stdout, cfg.LogJSON)
	chainPath := filepath.Join(cfg.DataDirectory, "sudharma-chain.json")
	statePath := filepath.Join(cfg.DataDirectory, "sudharma-state.json")
	mempoolPath := filepath.Join(cfg.DataDirectory, "sudharma-mempool.json")
	gpuActivationPath := filepath.Join(cfg.DataDirectory, "sudharma-gpu-v1-activation.json")
	chain, err := loadOrCreateChainWithGPUActivation(chainPath, gpuActivationPath, cfg.GPUV1ActivationHeight)
	if err != nil {
		return fmt.Errorf("blockchain startup failed: %w", err)
	}
	state, err := loadOrCreateState(chain, statePath)
	if err != nil {
		return fmt.Errorf("state startup failed: %w", err)
	}
	node, err := p2p.NewNode(cfg.NodeID, cfg.P2PAddress, chain.Height(), chain.Tip().Hash())
	if err != nil {
		return fmt.Errorf("P2P node creation failed: %w", err)
	}
	if err := node.SetChain(chain); err != nil {
		return fmt.Errorf("chain attachment failed: %w", err)
	}
	if err := node.SetState(state); err != nil {
		return fmt.Errorf("state attachment failed: %w", err)
	}
	if _, _, err := node.LoadMempoolFromFile(mempoolPath); err != nil && !os.IsNotExist(err) {
		log.Error("mempool_load_rejected", map[string]any{"error": err.Error()})
		node.Mempool().Clear()
	}
	if err := node.Start(); err != nil {
		return fmt.Errorf("P2P startup failed: %w", err)
	}
	nodeStopped := false
	defer func() {
		if !nodeStopped {
			_ = node.Stop()
		}
	}()
	for _, address := range cfg.Peers {
		peer, err := node.Connect(address)
		if err != nil {
			log.Error("peer_connect_failed", map[string]any{"peer": address, "error": err.Error()})
			continue
		}
		if err := node.SyncFromPeer(peer.NodeID, 10*time.Second); err != nil {
			log.Error("peer_chain_sync_failed", map[string]any{"peer": address, "error": err.Error()})
			continue
		}
		if err := node.SyncMempoolWithPeer(peer.NodeID); err != nil {
			log.Error("peer_mempool_sync_failed", map[string]any{"peer": address, "error": err.Error()})
		}
	}
	if err := saveData(chain, state, node, chainPath, statePath, mempoolPath); err != nil {
		return fmt.Errorf("startup persistence failed: %w", err)
	}
	rpcCfg := rpc.DefaultConfig()
	rpcCfg.ListenAddress = cfg.RPCAddress
	rpcCfg.EnableMetrics = cfg.Metrics
	rpcServer, err := rpc.NewServer(rpcCfg, node, chain, state)
	if err != nil {
		return fmt.Errorf("RPC server creation failed: %w", err)
	}
	rpcErrors := make(chan error, 1)
	go func() {
		e := rpcServer.ListenAndServe()
		if errors.Is(e, http.ErrServerClosed) {
			e = nil
		}
		rpcErrors <- e
	}()
	persistErrors := make(chan error, 1)
	stopPersist := make(chan struct{})
	if interval := cfg.PersistenceInterval(); interval > 0 {
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := saveData(chain, state, node, chainPath, statePath, mempoolPath); err != nil {
						select {
						case persistErrors <- err:
						default:
						}
					}
				case <-stopPersist:
					return
				}
			}
		}()
	}
	defer close(stopPersist)
	log.Info("node_started", map[string]any{"node_id": cfg.NodeID, "p2p": node.ListenAddress, "rpc": cfg.RPCAddress, "height": chain.Height(), "metrics": cfg.Metrics})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case sig := <-signals:
		log.Info("shutdown_signal", map[string]any{"signal": sig.String()})
	case err := <-rpcErrors:
		if err != nil {
			return fmt.Errorf("RPC server failed: %w", err)
		}
	case err := <-persistErrors:
		return fmt.Errorf("periodic persistence failed: %w", err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rpcServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("RPC shutdown failed: %w", err)
	}
	if err := saveData(chain, state, node, chainPath, statePath, mempoolPath); err != nil {
		return fmt.Errorf("shutdown persistence failed: %w", err)
	}
	if err := node.Stop(); err != nil {
		return fmt.Errorf("P2P shutdown failed: %w", err)
	}
	nodeStopped = true
	log.Info("node_stopped", map[string]any{"height": chain.Height()})
	return nil
}
func loadOrCreateChain(path string) (*blockchain.Chain, error) {
	if _, err := os.Stat(path); err == nil {
		return blockchain.LoadChainFromFile(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	c := blockchain.NewChain()
	if err := c.SaveToFile(path); err != nil {
		return nil, err
	}
	return c, nil
}
func loadOrCreateState(chain *blockchain.Chain, path string) (*blockchain.State, error) {
	if _, err := os.Stat(path); err == nil {
		return blockchain.LoadStateFromFile(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	s := blockchain.NewState()
	for h := uint64(1); h <= chain.Height(); h++ {
		b, ok := chain.BlockByHeight(h)
		if !ok {
			return nil, fmt.Errorf("missing block %d during state rebuild", h)
		}
		if b.MinerAddress == "" {
			return nil, fmt.Errorf("block %d has no miner address", h)
		}
		if _, err := blockchain.ProcessBlock(s, b, b.MinerAddress); err != nil {
			return nil, fmt.Errorf("replay block %d: %w", h, err)
		}
	}
	if err := s.SaveToFile(path); err != nil {
		return nil, err
	}
	return s, nil
}
func saveData(chain *blockchain.Chain, state *blockchain.State, node *p2p.Node, chainPath, statePath, mempoolPath string) error {
	if err := chain.SaveToFile(chainPath); err != nil {
		return fmt.Errorf("save chain: %w", err)
	}
	if err := state.SaveToFile(statePath); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	if err := node.SaveMempoolToFile(mempoolPath); err != nil {
		return fmt.Errorf("save mempool: %w", err)
	}
	return nil
}
