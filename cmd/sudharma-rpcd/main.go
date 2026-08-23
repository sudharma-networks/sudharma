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
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/rpc"
)

func main() {
	nodeID := flag.String("nodeid", "rpc-node", "unique Sudharma Network node ID")
	p2pAddress := flag.String("listen", "127.0.0.1:18444", "P2P listen address")
	rpcAddress := flag.String("rpc", rpc.DefaultListenAddress, "HTTP RPC listen address")
	peerAddress := flag.String("peer", "", "optional peer address to connect and synchronize with")
	dataDirectory := flag.String("datadir", "data-rpc-node", "node data directory")
	flag.Parse()

	chainPath := filepath.Join(*dataDirectory, "sudharma-chain.json")
	statePath := filepath.Join(*dataDirectory, "sudharma-state.json")
	mempoolPath := filepath.Join(*dataDirectory, "sudharma-mempool.json")

	chain, err := loadOrCreateChain(chainPath)
	if err != nil {
		fatal("blockchain startup failed", err)
	}
	state, err := loadOrCreateState(chain, statePath)
	if err != nil {
		fatal("state startup failed", err)
	}

	node, err := p2p.NewNode(*nodeID, *p2pAddress, chain.Height(), chain.Tip().Hash())
	if err != nil {
		fatal("P2P node creation failed", err)
	}
	if err := node.SetChain(chain); err != nil {
		fatal("chain attachment failed", err)
	}
	if err := node.SetState(state); err != nil {
		fatal("state attachment failed", err)
	}

	if _, _, err := node.LoadMempoolFromFile(mempoolPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("[RPCD] persisted mempool rejected: %v; continuing empty\n", err)
		node.Mempool().Clear()
	}

	if err := node.Start(); err != nil {
		fatal("P2P startup failed", err)
	}
	defer node.Stop()

	if *peerAddress != "" {
		peer, err := node.Connect(*peerAddress)
		if err != nil {
			fatal("peer bootstrap failed", err)
		}
		if err := node.SyncFromPeer(peer.NodeID, 10*time.Second); err != nil {
			fatal("chain synchronization failed", err)
		}
		if err := node.SyncMempoolWithPeer(peer.NodeID); err != nil {
			fatal("mempool synchronization failed", err)
		}
		if err := saveData(chain, state, node, chainPath, statePath, mempoolPath); err != nil {
			fatal("post-sync persistence failed", err)
		}
	}

	rpcConfig := rpc.DefaultConfig()
	rpcConfig.ListenAddress = *rpcAddress
	rpcServer, err := rpc.NewServer(rpcConfig, node, chain, state)
	if err != nil {
		fatal("RPC server creation failed", err)
	}

	rpcErrors := make(chan error, 1)
	go func() {
		if err := rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			rpcErrors <- err
			return
		}
		rpcErrors <- nil
	}()

	fmt.Printf("Sudharma RPC node running | P2P: %s | RPC: %s | Height: %d\n", node.ListenAddress, *rpcAddress, chain.Height())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case sig := <-signals:
		fmt.Printf("[RPCD] shutdown signal: %s\n", sig)
	case err := <-rpcErrors:
		if err != nil {
			fatal("RPC server failed", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rpcServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("[RPCD] RPC shutdown warning: %v\n", err)
	}
	if err := saveData(chain, state, node, chainPath, statePath, mempoolPath); err != nil {
		fatal("shutdown persistence failed", err)
	}
	if err := node.Stop(); err != nil {
		fmt.Printf("[RPCD] P2P shutdown warning: %v\n", err)
	}
	fmt.Println("Sudharma RPC node stopped cleanly.")
}

func loadOrCreateChain(path string) (*blockchain.Chain, error) {
	if _, err := os.Stat(path); err == nil {
		return blockchain.LoadChainFromFile(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	chain := blockchain.NewChain()
	if err := chain.SaveToFile(path); err != nil {
		return nil, err
	}
	return chain, nil
}

func loadOrCreateState(chain *blockchain.Chain, path string) (*blockchain.State, error) {
	if _, err := os.Stat(path); err == nil {
		return blockchain.LoadStateFromFile(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	state := blockchain.NewState()
	for height := uint64(1); height <= chain.Height(); height++ {
		block, ok := chain.BlockByHeight(height)
		if !ok {
			return nil, fmt.Errorf("missing block %d during state rebuild", height)
		}
		if block.MinerAddress == "" {
			return nil, fmt.Errorf("block %d has no miner address", height)
		}
		if _, err := blockchain.ProcessBlock(state, block, block.MinerAddress); err != nil {
			return nil, fmt.Errorf("replay block %d: %w", height, err)
		}
	}
	if err := state.SaveToFile(path); err != nil {
		return nil, err
	}
	return state, nil
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

func fatal(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}
