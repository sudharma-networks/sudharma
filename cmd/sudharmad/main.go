package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func main() {
	nodeID := flag.String(
		"nodeid",
		"node-a",
		"unique Sudharma Network node ID",
	)

	listenAddress := flag.String(
		"listen",
		"127.0.0.1:18444",
		"P2P listen address",
	)

	peerAddress := flag.String(
		"peer",
		"",
		"optional peer address to connect to",
	)

	dataDirectory := flag.String(
		"datadir",
		"data-node-a",
		"node data directory",
	)

	testTransaction := flag.Bool(
		"testtx",
		false,
		"broadcast one unfunded signed development transaction",
	)

	mineTest := flag.Bool(
		"minetest",
		false,
		"create a funded development transaction and mine it",
	)

	emptyBlocks := flag.Uint64(
		"emptyblocks",
		0,
		"mine N empty development blocks",
	)

	mineBlocks := flag.Uint64(
		"mineblocks",
		0,
		"mine N development blocks including valid mempool transactions",
	)

	testMinerAddress := flag.String(
		"testmineraddress",
		"",
		"reward address used with -emptyblocks or -mineblocks",
	)

	mempoolTest := flag.Bool(
		"mempooltest",
		false,
		"create one valid funded pending transaction and leave it in the mempool",
	)

	flag.Parse()

	fmt.Println("===================================")
	fmt.Println("          Sudharma Network Node")
	fmt.Println("===================================")

	chainPath := filepath.Join(
		*dataDirectory,
		"sudharma-chain.json",
	)

	statePath := filepath.Join(
		*dataDirectory,
		"sudharma-state.json",
	)

	mempoolPath := filepath.Join(
		*dataDirectory,
		"sudharma-mempool.json",
	)

	peersPath := filepath.Join(
		*dataDirectory,
		"sudharma-peers.json",
	)

	// =================================================
	// LOAD BLOCKCHAIN
	// =================================================

	var chain *blockchain.Chain

	loadedChain, err :=
		blockchain.LoadChainFromFile(
			chainPath,
		)

	if err == nil {
		chain = loadedChain

		fmt.Println(
			"Existing blockchain loaded.",
		)
	} else {
		chain =
			blockchain.NewChain()

		fmt.Println(
			"No existing blockchain found.",
		)

		fmt.Println(
			"Creating Sudharma Network genesis chain...",
		)

		if err :=
			chain.SaveToFile(
				chainPath,
			); err != nil {

			fmt.Println(
				"Failed to save genesis chain:",
				err,
			)

			return
		}
	}

	// =================================================
	// LOAD BLOCKCHAIN STATE
	// =================================================

	var nodeState *blockchain.State

	loadedState, stateErr :=
		blockchain.LoadStateFromFile(
			statePath,
		)

	if stateErr == nil {
		nodeState =
			loadedState

		fmt.Println(
			"Existing blockchain state loaded.",
		)

	} else if os.IsNotExist(stateErr) {
		// ------------------------------------------------
		// No state file exists yet.
		//
		// If the chain is still at genesis, create an
		// empty state.
		//
		// If blocks already exist, rebuild the state
		// deterministically by replaying those blocks.
		// ------------------------------------------------

		if chain.Height() == 0 {
			nodeState =
				blockchain.NewState()

			fmt.Println(
				"No existing blockchain state found.",
			)

			fmt.Println(
				"Creating Sudharma Network genesis state...",
			)
		} else {
			fmt.Println(
				"No blockchain state file found.",
			)

			fmt.Printf(
				"Rebuilding state from %d existing block(s)...\n",
				chain.Height(),
			)

			rebuiltState, err :=
				rebuildStateFromChain(
					chain,
				)

			if err != nil {
				fmt.Println(
					"Failed to rebuild blockchain state:",
					err,
				)

				fmt.Println()
				fmt.Println(
					"If this is an old development data directory",
				)

				fmt.Println(
					"created before persistent state/miner-address support,",
				)

				fmt.Println(
					"use a new empty -datadir for testing.",
				)

				return
			}

			nodeState =
				rebuiltState

			fmt.Println(
				"Blockchain state rebuilt successfully.",
			)
		}

		if err :=
			nodeState.SaveToFile(
				statePath,
			); err != nil {

			fmt.Println(
				"Failed to save initial blockchain state:",
				err,
			)

			return
		}

	} else {
		// A state file exists but cannot be decoded or
		// violates consensus rules.
		//
		// Do NOT silently replace a corrupted state.
		fmt.Println(
			"Failed to load blockchain state:",
			stateErr,
		)

		fmt.Println(
			"Refusing to start with invalid blockchain state.",
		)

		return
	}

	// =================================================
	// NODE INFORMATION
	// =================================================

	fmt.Println()

	fmt.Printf(
		"Coin:       %s (%s)\n",
		params.CoinName,
		params.CoinSymbol,
	)

	fmt.Printf(
		"Node ID:    %s\n",
		*nodeID,
	)

	fmt.Printf(
		"Listen:     %s\n",
		*listenAddress,
	)

	fmt.Printf(
		"Height:     %d\n",
		chain.Height(),
	)

	fmt.Printf(
		"Blocks:     %d\n",
		chain.Length(),
	)

	fmt.Printf(
		"Tip Hash:   %s\n",
		chain.Tip().Hash(),
	)

	fmt.Printf(
		"Total Work: %s\n",
		chain.TotalWork().String(),
	)

	fmt.Printf(
		"Chain File: %s\n",
		chainPath,
	)

	fmt.Printf(
		"State File: %s\n",
		statePath,
	)

	fmt.Printf(
		"Mempool File: %s\n",
		mempoolPath,
	)

	fmt.Printf(
		"Peers File: %s\n",
		peersPath,
	)

	fmt.Printf(
		"Treasury:   %s\n",
		nodeState.DevelopmentAddress(),
	)

	fmt.Printf(
		"Issued:     %.8f SUDH\n",
		float64(
			nodeState.IssuedSupply(),
		)/
			float64(
				params.CoinDecimals,
			),
	)

	fmt.Println(
		"State:      loaded and ready",
	)

	// =================================================
	// P2P NODE
	// =================================================

	networkNode, err :=
		p2p.NewNode(
			*nodeID,
			*listenAddress,
			chain.Height(),
			chain.Tip().Hash(),
		)

	if err != nil {
		fmt.Println(
			"Failed to create P2P node:",
			err,
		)

		return
	}

	// Attach blockchain.
	if err :=
		networkNode.SetChain(
			chain,
		); err != nil {

		fmt.Println(
			"Failed to attach blockchain:",
			err,
		)

		return
	}

	// Attach confirmed blockchain state.
	if err :=
		networkNode.SetState(
			nodeState,
		); err != nil {

		fmt.Println(
			"Failed to attach blockchain state:",
			err,
		)

		return
	}

	// =================================================
	// LOAD + REVALIDATE PERSISTED MEMPOOL
	// =================================================

	loadedPending,
		rejectedPending,
		mempoolErr :=
		networkNode.LoadMempoolFromFile(
			mempoolPath,
		)

	if mempoolErr != nil {
		fmt.Println(
			"Failed to load persisted mempool:",
			mempoolErr,
		)

		fmt.Println(
			"Continuing with an empty mempool.",
		)

		networkNode.Mempool().Clear()
	} else {
		fmt.Printf(
			"Mempool restored: %d transaction(s), rejected: %d\n",
			loadedPending,
			rejectedPending,
		)
	}

	// =================================================
	// LOAD PERSISTED KNOWN PEERS
	// =================================================

	knownPeers, peerLoadErr :=
		p2p.LoadKnownPeersFromFile(
			peersPath,
			*nodeID,
		)

	if peerLoadErr != nil {
		fmt.Println(
			"Failed to load persisted known peers:",
			peerLoadErr,
		)

		fmt.Println(
			"Continuing with an empty known-peer list.",
		)

		knownPeers = []p2p.KnownPeer{}
	} else {
		fmt.Printf(
			"Known peers restored: %d\n",
			len(knownPeers),
		)
	}

	if err :=
		networkNode.Start(); err != nil {

		fmt.Println(
			"Failed to start P2P node:",
			err,
		)

		return
	}

	defer networkNode.Stop()

	fmt.Println()
	fmt.Println(
		"P2P networking started.",
	)

	fmt.Printf(
		"Listening on %s\n",
		networkNode.ListenAddress,
	)

	var remoteNodeID string

	// =================================================
	// AUTOMATIC RECONNECT TO REMEMBERED PEERS
	// =================================================
	//
	// Automatic reconnect is used only when the user did not
	// explicitly supply -peer. Offline remembered peers are
	// non-fatal: the node continues starting and can reconnect
	// on a later restart.
	if *peerAddress == "" && len(knownPeers) > 0 {

		fmt.Println()
		fmt.Printf(
			"[AUTO-PEER] Attempting %d remembered peer connection(s)...\n",
			len(knownPeers),
		)

		for _, knownPeer := range knownPeers {

			if knownPeer.NodeID == "" ||
				knownPeer.Address == "" {

				continue
			}

			fmt.Printf(
				"[AUTO-PEER] Reconnecting to %s at %s...\n",
				knownPeer.NodeID,
				knownPeer.Address,
			)

			peer, err :=
				networkNode.Connect(
					knownPeer.Address,
				)

			if err != nil {
				fmt.Printf(
					"[AUTO-PEER] Reconnect failed for %s: %v\n",
					knownPeer.NodeID,
					err,
				)

				continue
			}

			remoteNodeID =
				peer.NodeID

			fmt.Printf(
				"[AUTO-PEER] Connected successfully: %s (%s)\n",
				peer.NodeID,
				knownPeer.Address,
			)

			// Refresh the remembered identity/address using the
			// peer's current handshake information.
			peerAddressToRemember :=
				peer.ListenAddress

			if peerAddressToRemember == "" {
				peerAddressToRemember =
					knownPeer.Address
			}

			knownPeers = rememberKnownPeer(
				knownPeers,
				p2p.KnownPeer{
					NodeID:  peer.NodeID,
					Address: peerAddressToRemember,
				},
			)

			// Merge dynamically discovered peers into the persistent

			// known-peer list before saving on shutdown.

			for _, discoveredPeer := range networkNode.DiscoveredPeersSnapshot() {

				knownPeers = rememberKnownPeer(

					knownPeers,

					discoveredPeer,
				)

			}

			if err :=
				p2p.SaveKnownPeersToFile(
					peersPath,
					*nodeID,
					knownPeers,
				); err != nil {

				fmt.Printf(
					"[AUTO-PEER] Failed saving peer database: %v\n",
					err,
				)
			}

			fmt.Printf(
				"[AUTO-PEER] Evaluating chain with %s...\n",
				peer.NodeID,
			)

			if err :=
				networkNode.SyncFromPeer(
					peer.NodeID,
					10*time.Second,
				); err != nil {

				fmt.Printf(
					"[AUTO-PEER] Chain synchronization with %s failed: %v\n",
					peer.NodeID,
					err,
				)

				continue
			}

			if err :=
				networkNode.SyncMempoolWithPeer(
					peer.NodeID,
				); err != nil {

				fmt.Printf(
					"[AUTO-PEER] Mempool synchronization with %s failed: %v\n",
					peer.NodeID,
					err,
				)

				continue
			}

			if err :=
				saveNodeData(
					chain,
					nodeState,
					chainPath,
					statePath,
				); err != nil {

				fmt.Printf(
					"[AUTO-PEER] Failed saving synchronized chain/state: %v\n",
					err,
				)

				continue
			}

			if err :=
				networkNode.SaveMempoolToFile(
					mempoolPath,
				); err != nil {

				fmt.Printf(
					"[AUTO-PEER] Failed saving synchronized mempool: %v\n",
					err,
				)

				continue
			}

			fmt.Printf(
				"[AUTO-PEER] Synchronization complete with %s | Height: %d | Work: %s\n",
				peer.NodeID,
				chain.Height(),
				chain.TotalWork().String(),
			)
		}
	}

	// =================================================
	// OPTIONAL PEER CONNECTION
	// =================================================

	if *peerAddress != "" {
		fmt.Printf(
			"Connecting to peer %s...\n",
			*peerAddress,
		)

		peer, err :=
			networkNode.Connect(
				*peerAddress,
			)

		if err != nil {
			fmt.Println(
				"Peer connection failed:",
				err,
			)

		} else {
			remoteNodeID =
				peer.NodeID

			fmt.Println(
				"Peer connected successfully!",
			)

			fmt.Printf(
				"Remote Node ID: %s\n",
				peer.NodeID,
			)

			fmt.Printf(
				"Remote Height:  %d\n",
				peer.Height,
			)

			fmt.Printf(
				"Remote Tip:     %s\n",
				peer.TipHash,
			)

			fmt.Printf(
				"Remote Work:    %s\n",
				peer.TotalWork,
			)

			fmt.Printf(
				"Local Work:     %s\n",
				chain.TotalWork().String(),
			)

			peerAddressToRemember :=
				peer.ListenAddress

			if peerAddressToRemember == "" {
				peerAddressToRemember =
					*peerAddress
			}

			knownPeers = rememberKnownPeer(
				knownPeers,
				p2p.KnownPeer{
					NodeID:  peer.NodeID,
					Address: peerAddressToRemember,
				},
			)

			// Merge dynamically discovered peers into the persistent

			// known-peer list before saving on shutdown.

			for _, discoveredPeer := range networkNode.DiscoveredPeersSnapshot() {

				knownPeers = rememberKnownPeer(

					knownPeers,

					discoveredPeer,
				)

			}

			if err :=
				p2p.SaveKnownPeersToFile(
					peersPath,
					*nodeID,
					knownPeers,
				); err != nil {

				fmt.Println(
					"Failed to save known peer:",
					err,
				)
			} else {
				fmt.Printf(
					"Known peer saved: %s (%s)\n",
					peer.NodeID,
					peerAddressToRemember,
				)
			}

			// =============================================
			// AUTOMATIC WORK-BASED SYNCHRONIZATION
			// =============================================
			//
			// Always ask SyncFromPeer to compare the chains.
			// SyncFromPeer now uses cumulative Proof-of-Work,
			// not height alone, and safely handles:
			//
			//   * ordinary forward synchronization
			//   * same-height competing forks
			//   * shorter chains with greater total work
			//   * weaker peers that should be ignored
			//
			// It independently downloads and validates a
			// candidate before any reorganization is adopted.

			fmt.Println()
			fmt.Println(
				"[SYNC] Evaluating peer chain by cumulative work...",
			)

			if err :=
				networkNode.SyncFromPeer(
					peer.NodeID,
					10*time.Second,
				); err != nil {

				fmt.Println(
					"[SYNC] Automatic synchronization failed:",
					err,
				)

				return
			}

			// Chain/state synchronization is complete.
			// Only now exchange pending transactions so
			// funding balances/nonces are already available.
			if err :=
				networkNode.SyncMempoolWithPeer(
					peer.NodeID,
				); err != nil {

				fmt.Println(
					"[MEMPOOL] Post-sync mempool synchronization failed:",
					err,
				)

				return
			}

			fmt.Printf(
				"[SYNC] Chain evaluation complete | Height: %d | Work: %s\n",
				chain.Height(),
				chain.TotalWork().String(),
			)

			if err :=
				networkNode.SendGetPeers(
					peer.NodeID,
				); err != nil {

				fmt.Println(
					"[PEERS] Peer discovery request failed:",
					err,
				)
			} else {
				fmt.Printf(
					"[PEERS] Requested peer list from %s\n",
					peer.NodeID,
				)
			}
			// Persist immediately after sync/reorg evaluation.
			// If the local chain was kept, this is harmless.
			// If a better peer chain was adopted, both chain
			// and rebuilt state are now safely persisted.
			if err :=
				saveNodeData(
					chain,
					nodeState,
					chainPath,
					statePath,
				); err != nil {

				fmt.Println(
					"[SYNC] Failed to save node data:",
					err,
				)

				return
			}

			if err :=
				networkNode.SaveMempoolToFile(
					mempoolPath,
				); err != nil {

				fmt.Println(
					"[SYNC] Failed to save mempool:",
					err,
				)

				return
			}

			fmt.Println(
				"[SYNC] Blockchain, state and mempool saved.",
			)
		}
	}

	fmt.Printf(
		"Active Peers: %d\n",
		networkNode.PeerCount(),
	)

	fmt.Printf(
		"Mempool:     %d transaction(s)\n",
		networkNode.MempoolCount(),
	)

	// =================================================
	// UNFUNDED TRANSACTION NETWORK TEST
	// =================================================

	if *testTransaction {
		if networkNode.PeerCount() == 0 {
			fmt.Println(
				"[TEST TX] No connected peer.",
			)
		} else {
			runUnfundedTransactionTest(
				networkNode,
			)
		}
	}

	// =================================================
	// LOCAL MINING TEST
	// =================================================

	if *mineTest {
		if err :=
			runMiningTest(
				chain,
				nodeState,
				networkNode,
				chainPath,
				statePath,
			); err != nil {

			fmt.Println(
				"Mining test failed:",
				err,
			)

			return
		}
	}

	// =================================================
	// LIVE MEMPOOL PERSISTENCE TEST
	// =================================================

	if *mempoolTest {
		if err :=
			runMempoolPersistenceTest(
				chain,
				nodeState,
				networkNode,
				chainPath,
				statePath,
				mempoolPath,
			); err != nil {

			fmt.Println(
				"Mempool persistence test failed:",
				err,
			)

			return
		}
	}

	// =================================================
	// EMPTY-BLOCK DEVELOPMENT MINING
	// =================================================

	if *emptyBlocks > 0 {
		if *testMinerAddress == "" {
			fmt.Println(
				"-testmineraddress is required when -emptyblocks is used",
			)
			return
		}

		if err :=
			runEmptyBlockMiningTest(
				chain,
				nodeState,
				networkNode,
				chainPath,
				statePath,
				*testMinerAddress,
				*emptyBlocks,
			); err != nil {

			fmt.Println(
				"Empty-block mining test failed:",
				err,
			)

			return
		}
	}

	// =================================================
	// TRANSACTION-CONFIRMING DEVELOPMENT MINING
	// =================================================

	if *mineBlocks > 0 && *testMinerAddress == "" {
		fmt.Println(
			"-testmineraddress is required when -mineblocks is used",
		)
		return
	}

	exitNodeLoop, err := runBlockMiningMode(
		*mineBlocks,
		func() error {
			return runBlockMiningTest(
				chain,
				nodeState,
				networkNode,
				chainPath,
				statePath,
				*testMinerAddress,
				*mineBlocks,
			)
		},
	)
	if err != nil {
		fmt.Println(
			"Block mining test failed:",
			err,
		)

		return
	}
	if exitNodeLoop {
		return
	}

	// =================================================
	// MAIN NODE LOOP
	// =================================================

	fmt.Println()
	fmt.Println(
		"Sudharma Network node is running.",
	)

	fmt.Println(
		"Press Ctrl+C to stop safely.",
	)

	stopChannel :=
		make(
			chan os.Signal,
			1,
		)

	signal.Notify(
		stopChannel,
		os.Interrupt,
		syscall.SIGTERM,
	)

	ticker :=
		time.NewTicker(
			10 * time.Second,
		)

	defer ticker.Stop()

	running := true

	for running {
		select {

		case <-ticker.C:
			if err :=
				networkNode.SaveMempoolToFile(
					mempoolPath,
				); err != nil {

				fmt.Printf(
					"[MEMPOOL] Periodic save failed: %v\n",
					err,
				)
			}

			if reconnectedNodeID :=
				reconnectKnownPeersRuntime(
					networkNode,
					knownPeers,
					chain,
					nodeState,
					chainPath,
					statePath,
					mempoolPath,
				); reconnectedNodeID != "" {

				remoteNodeID =
					reconnectedNodeID
			}

			fmt.Printf(
				"[NODE] Height: %d | Peers: %d | Mempool: %d | Issued: %.8f SUDH\n",
				chain.Height(),
				networkNode.PeerCount(),
				networkNode.MempoolCount(),
				float64(
					nodeState.IssuedSupply(),
				)/
					float64(
						params.CoinDecimals,
					),
			)

			if remoteNodeID != "" &&
				networkNode.IsPeerConnected(remoteNodeID) {
				nonce :=
					uint64(
						time.Now().UnixNano(),
					)

				if err :=
					networkNode.SendPing(
						remoteNodeID,
						nonce,
					); err != nil {

					fmt.Printf(
						"[P2P] Ping failed: %v\n",
						err,
					)
				} else {
					fmt.Printf(
						"[P2P] Ping sent to %s\n",
						remoteNodeID,
					)
				}
			}

		case <-stopChannel:
			running = false
		}
	}

	// =================================================
	// SAFE SHUTDOWN
	// =================================================

	fmt.Println()
	fmt.Println(
		"Stopping Sudharma Network node...",
	)

	if err :=
		saveNodeData(
			chain,
			nodeState,
			chainPath,
			statePath,
		); err != nil {

		fmt.Println(
			"Failed to save Sudharma Network node data:",
			err,
		)

		return
	}

	fmt.Println(
		"Blockchain saved successfully.",
	)

	fmt.Println(
		"Blockchain state saved successfully.",
	)

	if err :=
		networkNode.SaveMempoolToFile(
			mempoolPath,
		); err != nil {

		fmt.Println(
			"Failed to save Sudharma Network mempool:",
			err,
		)

		return
	}

	fmt.Println(
		"Mempool saved successfully.",
	)

	// Merge dynamically discovered peers into the persistent

	// known-peer list before saving on shutdown.

	for _, discoveredPeer := range networkNode.DiscoveredPeersSnapshot() {

		knownPeers = rememberKnownPeer(

			knownPeers,

			discoveredPeer,
		)

	}

	if err :=
		p2p.SaveKnownPeersToFile(
			peersPath,
			*nodeID,
			knownPeers,
		); err != nil {

		fmt.Println(
			"Failed to save Sudharma Network known peers:",
			err,
		)

		return
	}

	fmt.Println(
		"Known peers saved successfully.",
	)

	fmt.Println(
		"Sudharma Network node stopped.",
	)
}

// reconnectKnownPeersRuntime checks remembered peers while the node
// is already running. Missing peers are retried without restarting
// Sudharma Network. A peer that is still offline is non-fatal.
func reconnectKnownPeersRuntime(
	networkNode *p2p.Node,
	knownPeers []p2p.KnownPeer,
	chain *blockchain.Chain,
	nodeState *blockchain.State,
	chainPath string,
	statePath string,
	mempoolPath string,
) string {

	if networkNode == nil ||
		chain == nil ||
		nodeState == nil {

		return ""
	}

	var lastReconnectedNodeID string

	for _, knownPeer := range knownPeers {

		if knownPeer.NodeID == "" ||
			knownPeer.Address == "" {

			continue
		}

		if networkNode.IsPeerConnected(
			knownPeer.NodeID,
		) {

			continue
		}

		fmt.Printf(
			"[RECONNECT] Peer %s is offline. Reconnecting to %s...\n",
			knownPeer.NodeID,
			knownPeer.Address,
		)

		peer, err :=
			networkNode.Connect(
				knownPeer.Address,
			)

		if err != nil {
			fmt.Printf(
				"[RECONNECT] %s still unavailable: %v\n",
				knownPeer.NodeID,
				err,
			)

			continue
		}

		lastReconnectedNodeID =
			peer.NodeID

		fmt.Printf(
			"[RECONNECT] Connected successfully: %s (%s)\n",
			peer.NodeID,
			knownPeer.Address,
		)

		if err :=
			networkNode.SyncFromPeer(
				peer.NodeID,
				10*time.Second,
			); err != nil {

			fmt.Printf(
				"[RECONNECT] Chain synchronization with %s failed: %v\n",
				peer.NodeID,
				err,
			)

			continue
		}

		if err :=
			networkNode.SyncMempoolWithPeer(
				peer.NodeID,
			); err != nil {

			fmt.Printf(
				"[RECONNECT] Mempool synchronization with %s failed: %v\n",
				peer.NodeID,
				err,
			)

			continue
		}

		if err :=
			saveNodeData(
				chain,
				nodeState,
				chainPath,
				statePath,
			); err != nil {

			fmt.Printf(
				"[RECONNECT] Failed saving synchronized chain/state: %v\n",
				err,
			)

			continue
		}

		if err :=
			networkNode.SaveMempoolToFile(
				mempoolPath,
			); err != nil {

			fmt.Printf(
				"[RECONNECT] Failed saving synchronized mempool: %v\n",
				err,
			)

			continue
		}

		fmt.Printf(
			"[RECONNECT] Recovery complete with %s | Height: %d | Work: %s\n",
			peer.NodeID,
			chain.Height(),
			chain.TotalWork().String(),
		)
	}

	return lastReconnectedNodeID
}

// rememberKnownPeer adds or updates one known peer in memory.
// Node ID is the stable identity key. If the peer advertises a new
// address later, the stored address is updated.
func rememberKnownPeer(
	peers []p2p.KnownPeer,
	candidate p2p.KnownPeer,
) []p2p.KnownPeer {

	for i := range peers {
		if peers[i].NodeID == candidate.NodeID {
			peers[i] = candidate
			return peers
		}
	}

	return append(
		peers,
		candidate,
	)
}

// rebuildStateFromChain reconstructs confirmed state
// from every non-genesis block in the blockchain.
func rebuildStateFromChain(
	chain *blockchain.Chain,
) (*blockchain.State, error) {

	if chain == nil {
		return nil,
			fmt.Errorf(
				"chain cannot be nil",
			)
	}

	state :=
		blockchain.NewState()

	for height := uint64(1); height <= chain.Height(); height++ {

		block, ok :=
			chain.BlockByHeight(
				height,
			)

		if !ok {
			return nil,
				fmt.Errorf(
					"block %d is missing",
					height,
				)
		}

		if block.MinerAddress == "" {
			return nil,
				fmt.Errorf(
					"block %d has no miner address",
					height,
				)
		}

		if _, err :=
			blockchain.ProcessBlock(
				state,
				block,
				block.MinerAddress,
			); err != nil {

			return nil,
				fmt.Errorf(
					"failed replaying block %d: %w",
					height,
					err,
				)
		}
	}

	return state, nil
}

// saveNodeData persists both the blockchain
// and the confirmed blockchain state.
func saveNodeData(
	chain *blockchain.Chain,
	state *blockchain.State,
	chainPath string,
	statePath string,
) error {

	if chain == nil {
		return fmt.Errorf(
			"chain cannot be nil",
		)
	}

	if state == nil {
		return fmt.Errorf(
			"state cannot be nil",
		)
	}

	if err :=
		chain.SaveToFile(
			chainPath,
		); err != nil {

		return fmt.Errorf(
			"failed saving blockchain: %w",
			err,
		)
	}

	if err :=
		state.SaveToFile(
			statePath,
		); err != nil {

		return fmt.Errorf(
			"failed saving blockchain state: %w",
			err,
		)
	}

	return nil
}

func runUnfundedTransactionTest(
	networkNode *p2p.Node,
) {

	sender, err :=
		wallet.NewWallet()

	if err != nil {
		fmt.Println(err)
		return
	}

	receiver, err :=
		wallet.NewWallet()

	if err != nil {
		fmt.Println(err)
		return
	}

	amount :=
		uint64(10) *
			params.CoinDecimals

	tx :=
		transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			amount,
			1,
		)

	if err :=
		tx.Sign(
			sender,
		); err != nil {

		fmt.Println(err)
		return
	}

	fmt.Println()
	fmt.Println(
		"========== TEST TRANSACTION ==========",
	)

	fmt.Printf(
		"Transaction ID: %s\n",
		tx.ID,
	)

	fmt.Printf(
		"From:           %s\n",
		tx.From,
	)

	fmt.Printf(
		"To:             %s\n",
		tx.To,
	)

	fmt.Println(
		"Amount:         10 SUDH",
	)

	fmt.Printf(
		"Nonce:          %d\n",
		tx.Nonce,
	)

	fmt.Println(
		"Signature:      VALID",
	)

	if err :=
		networkNode.BroadcastTransaction(
			tx,
		); err != nil {

		fmt.Println(
			"Broadcast failed:",
			err,
		)
	} else {
		fmt.Println(
			"Broadcast:      SUCCESS",
		)
	}

	fmt.Println(
		"=====================================",
	)
}

func runMiningTest(
	chain *blockchain.Chain,
	nodeState *blockchain.State,
	networkNode *p2p.Node,
	chainPath string,
	statePath string,
) error {

	fmt.Println()
	fmt.Println(
		"========== SUDHARMA NETWORK MINING TEST ==========",
	)

	// -------------------------------------------------
	// Create miner and receiver.
	//
	// IMPORTANT:
	// No artificial state.Credit() funding is used.
	// Every spendable SUDH must originate from an
	// on-chain block subsidy.
	// -------------------------------------------------

	minerWallet, err :=
		wallet.NewWallet()

	if err != nil {
		return fmt.Errorf(
			"failed to create miner wallet: %w",
			err,
		)
	}

	receiver, err :=
		wallet.NewWallet()

	if err != nil {
		return fmt.Errorf(
			"failed to create receiver wallet: %w",
			err,
		)
	}

	pool :=
		networkNode.Mempool()

	// =================================================
	// BLOCK 1 Î“Ã²Â¼â”œâ”¤Î“Ã¶Â£â”œÂºÎ“Ã¶Â£Î“Ã²Ã³ MINE EMPTY BLOCK TO CREATE REAL FUNDS
	// =================================================

	fmt.Printf(
		"Mining funding block #%d...\n",
		chain.Height()+1,
	)

	firstResult, firstReward, err :=
		miner.MineNextBlock(
			chain,
			nodeState,
			pool,
			minerWallet.Address,
			1_000_000,
		)

	if err != nil {
		return fmt.Errorf(
			"failed to mine funding block: %w",
			err,
		)
	}

	if !firstResult.Found {
		return fmt.Errorf(
			"funding block was not found",
		)
	}

	fmt.Println(
		"Funding block found!",
	)

	// Refresh the P2P-advertised height and tip after mining.
	networkNode.RefreshChainStatus()

	fmt.Printf(
		"Height:        %d\n",
		chain.Height(),
	)

	fmt.Printf(
		"Hash:          %s\n",
		firstResult.Hash,
	)

	fmt.Printf(
		"Miner Reward:  %.8f SUDH\n",
		float64(firstReward)/
			float64(params.CoinDecimals),
	)

	fmt.Printf(
		"Miner Balance: %.8f SUDH\n",
		float64(
			nodeState.Balance(
				minerWallet.Address,
			),
		)/
			float64(params.CoinDecimals),
	)

	// The miner now has real blockchain-created SUDH
	// and can legally spend it.

	// =================================================
	// CREATE TRANSACTION FOR BLOCK 2
	// =================================================

	amount :=
		uint64(10) *
			params.CoinDecimals

	expectedNonce, err :=
		nodeState.ExpectedNonce(
			minerWallet.Address,
		)

	if err != nil {
		return fmt.Errorf(
			"failed to determine transaction nonce: %w",
			err,
		)
	}

	tx :=
		transactions.NewTransaction(
			minerWallet.Address,
			receiver.Address,
			amount,
			expectedNonce,
		)

	if err :=
		tx.Sign(
			minerWallet,
		); err != nil {

		return fmt.Errorf(
			"failed to sign transaction: %w",
			err,
		)
	}

	if err :=
		blockchain.ValidateMempoolTransaction(
			nodeState,
			pool.AllTransactions(),
			tx,
		); err != nil {

		return fmt.Errorf(
			"mempool validation failed: %w",
			err,
		)
	}

	if err :=
		pool.AddTransaction(
			tx,
		); err != nil {

		return fmt.Errorf(
			"failed to add transaction to mempool: %w",
			err,
		)
	}

	fmt.Println()
	fmt.Printf(
		"Transaction added to mempool: %s\n",
		tx.ID,
	)

	fmt.Printf(
		"From miner:     %s\n",
		minerWallet.Address,
	)

	fmt.Printf(
		"To receiver:    %s\n",
		receiver.Address,
	)

	fmt.Println(
		"Amount:         10.00000000 SUDH",
	)

	fmt.Printf(
		"Nonce:          %d\n",
		tx.Nonce,
	)

	fmt.Printf(
		"Mempool before mining: %d\n",
		pool.Count(),
	)

	// =================================================
	// BLOCK 2 Î“Ã²Â¼â”œâ”¤Î“Ã¶Â£â”œÂºÎ“Ã¶Â£Î“Ã²Ã³ MINE REAL TRANSACTION
	// =================================================

	fmt.Printf(
		"Mining transaction block #%d...\n",
		chain.Height()+1,
	)

	secondResult, secondReward, err :=
		miner.MineNextBlock(
			chain,
			nodeState,
			pool,
			minerWallet.Address,
			1_000_000,
		)

	if err != nil {
		return fmt.Errorf(
			"failed to mine transaction block: %w",
			err,
		)
	}

	if !secondResult.Found {
		return fmt.Errorf(
			"transaction block was not found",
		)
	}

	fmt.Println(
		"Transaction block found!",
	)

	// Refresh the P2P-advertised height and tip after mining.
	networkNode.RefreshChainStatus()

	fmt.Printf(
		"Height:      %d\n",
		chain.Height(),
	)

	fmt.Printf(
		"Hash:        %s\n",
		secondResult.Hash,
	)

	fmt.Printf(
		"Nonce:       %d\n",
		secondResult.Nonce,
	)

	fmt.Printf(
		"Hashes Run:  %d\n",
		secondResult.HashesRun,
	)

	fmt.Printf(
		"Mining Time: %s\n",
		secondResult.Duration,
	)

	fmt.Printf(
		"Block Reward + Miner Fee: %.8f SUDH\n",
		float64(secondReward)/
			float64(params.CoinDecimals),
	)

	// =================================================
	// FINAL CONSENSUS STATE
	// =================================================

	fmt.Println()
	fmt.Println(
		"----- FINAL ON-CHAIN STATE -----",
	)

	fmt.Printf(
		"Miner Balance:       %.8f SUDH\n",
		float64(
			nodeState.Balance(
				minerWallet.Address,
			),
		)/
			float64(params.CoinDecimals),
	)

	fmt.Printf(
		"Receiver Balance:    %.8f SUDH\n",
		float64(
			nodeState.Balance(
				receiver.Address,
			),
		)/
			float64(params.CoinDecimals),
	)

	fmt.Printf(
		"Development Treasury: %.8f SUDH\n",
		float64(
			nodeState.Balance(
				params.DevelopmentTreasuryAddress,
			),
		)/
			float64(params.CoinDecimals),
	)

	fmt.Printf(
		"Issued Supply:       %.8f SUDH\n",
		float64(
			nodeState.IssuedSupply(),
		)/
			float64(params.CoinDecimals),
	)

	fmt.Printf(
		"Mempool:             %d transaction(s)\n",
		pool.Count(),
	)

	fmt.Println(
		"--------------------------------",
	)

	// =================================================
	// SAVE VALID CHAIN + STATE
	// =================================================

	if err :=
		saveNodeData(
			chain,
			nodeState,
			chainPath,
			statePath,
		); err != nil {

		return err
	}

	fmt.Println(
		"Blockchain and state saved.",
	)

	fmt.Println(
		"===========================================",
	)

	return nil
}

// runEmptyBlockMiningTest mines development blocks containing no
// transactions and pays every block subsidy to minerAddress.
//
// This helper exists specifically for live fork/reorg testing.
// It lets two independent chains share the same funded address
// without needing the winning node to possess that address's
// private key.
func runEmptyBlockMiningTest(
	chain *blockchain.Chain,
	nodeState *blockchain.State,
	networkNode *p2p.Node,
	chainPath string,
	statePath string,
	minerAddress string,
	count uint64,
) error {

	if chain == nil {
		return fmt.Errorf(
			"chain cannot be nil",
		)
	}

	if nodeState == nil {
		return fmt.Errorf(
			"state cannot be nil",
		)
	}

	if networkNode == nil {
		return fmt.Errorf(
			"network node cannot be nil",
		)
	}

	if minerAddress == "" {
		return fmt.Errorf(
			"miner address cannot be empty",
		)
	}

	if count == 0 {
		return fmt.Errorf(
			"empty block count must be greater than zero",
		)
	}

	if count > 100 {
		return fmt.Errorf(
			"development empty block count cannot exceed 100",
		)
	}

	pool :=
		networkNode.Mempool()

	if pool.Count() != 0 {
		return fmt.Errorf(
			"cannot mine empty test blocks while mempool contains %d transaction(s)",
			pool.Count(),
		)
	}

	fmt.Println()
	fmt.Println(
		"========== SUDHARMA NETWORK EMPTY-BLOCK TEST ==========",
	)

	fmt.Printf(
		"Reward Address: %s\n",
		minerAddress,
	)

	fmt.Printf(
		"Blocks to Mine: %d\n",
		count,
	)

	for i := uint64(0); i < count; i++ {
		targetHeight :=
			chain.Height() + 1

		fmt.Printf(
			"Mining empty block #%d...\n",
			targetHeight,
		)

		result,
			reward,
			err :=
			miner.MineNextBlock(
				chain,
				nodeState,
				pool,
				minerAddress,
				1_000_000,
			)

		if err != nil {
			return fmt.Errorf(
				"failed mining empty block %d: %w",
				targetHeight,
				err,
			)
		}

		if !result.Found {
			return fmt.Errorf(
				"empty block %d was not found",
				targetHeight,
			)
		}

		networkNode.RefreshChainStatus()

		if err :=
			networkNode.BroadcastBlock(
				result.Block,
			); err != nil {

			return fmt.Errorf(
				"failed broadcasting empty block %d: %w",
				targetHeight,
				err,
			)
		}

		fmt.Printf(
			"[BLOCK] Broadcast newly mined block #%d to connected peer(s)\n",
			chain.Height(),
		)

		fmt.Printf(
			"Block #%d found | Hash: %s | Reward: %.8f SUDH | Work: %s\n",
			chain.Height(),
			result.Hash,
			float64(reward)/float64(params.CoinDecimals),
			chain.TotalWork().String(),
		)
	}

	if err :=
		saveNodeData(
			chain,
			nodeState,
			chainPath,
			statePath,
		); err != nil {

		return err
	}

	fmt.Println()
	fmt.Printf(
		"Final Height: %d\n",
		chain.Height(),
	)

	fmt.Printf(
		"Final Work:   %s\n",
		chain.TotalWork().String(),
	)

	fmt.Printf(
		"Reward Balance: %.8f SUDH\n",
		float64(
			nodeState.Balance(
				minerAddress,
			),
		)/
			float64(
				params.CoinDecimals,
			),
	)

	fmt.Printf(
		"Issued Supply: %.8f SUDH\n",
		float64(
			nodeState.IssuedSupply(),
		)/
			float64(
				params.CoinDecimals,
			),
	)

	fmt.Println(
		"Blockchain and state saved.",
	)

	fmt.Println(
		"================================================",
	)

	return nil
}

// runBlockMiningMode runs positive -mineblocks requests as one-shot work and
// reports whether main must exit instead of entering the normal node loop.
func runBlockMiningMode(count uint64, mine func() error) (exitNodeLoop bool, err error) {
	if count == 0 {
		return false, nil
	}
	if mine == nil {
		return true, fmt.Errorf("block mining function is required")
	}
	return true, mine()
}

// runBlockMiningTest mines and broadcasts a bounded number of development
// blocks using the node's current mempool. Unlike runEmptyBlockMiningTest, this
// helper intentionally permits pending transactions so testnet operators can
// confirm an end-to-end transaction without changing consensus behavior.
func runBlockMiningTest(
	chain *blockchain.Chain,
	nodeState *blockchain.State,
	networkNode *p2p.Node,
	chainPath string,
	statePath string,
	minerAddress string,
	count uint64,
) error {
	if chain == nil {
		return fmt.Errorf("chain cannot be nil")
	}
	if nodeState == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if networkNode == nil {
		return fmt.Errorf("network node cannot be nil")
	}
	if minerAddress == "" {
		return fmt.Errorf("miner address cannot be empty")
	}
	if count == 0 {
		return fmt.Errorf("block count must be greater than zero")
	}
	if count > 100 {
		return fmt.Errorf("development block count cannot exceed 100")
	}

	pool := networkNode.Mempool()

	fmt.Println()
	fmt.Println("========== SUDHARMA NETWORK BLOCK MINING TEST ==========")
	fmt.Printf("Reward Address: %s\n", minerAddress)
	fmt.Printf("Blocks to Mine: %d\n", count)
	fmt.Printf("Pending Transactions: %d\n", pool.Count())

	for i := uint64(0); i < count; i++ {
		targetHeight := chain.Height() + 1
		pending := pool.Count()
		fmt.Printf("Mining block #%d with %d pending transaction(s)...\n", targetHeight, pending)

		result, reward, err := miner.MineNextBlock(
			chain,
			nodeState,
			pool,
			minerAddress,
			1_000_000,
		)
		if err != nil {
			return fmt.Errorf("failed mining block %d: %w", targetHeight, err)
		}
		if !result.Found {
			return fmt.Errorf("block %d was not found", targetHeight)
		}

		networkNode.RefreshChainStatus()
		if err := networkNode.BroadcastBlock(result.Block); err != nil {
			return fmt.Errorf("failed broadcasting block %d: %w", targetHeight, err)
		}

		fmt.Printf(
			"Block #%d found | Hash: %s | Transactions: %d | Reward: %.8f SUDH | Work: %s\n",
			chain.Height(),
			result.Hash,
			len(result.Block.Transactions),
			float64(reward)/float64(params.CoinDecimals),
			chain.TotalWork().String(),
		)
	}

	if err := saveNodeData(chain, nodeState, chainPath, statePath); err != nil {
		return err
	}

	fmt.Printf("Final Height: %d\n", chain.Height())
	fmt.Printf("Final Work:   %s\n", chain.TotalWork().String())
	fmt.Printf("Remaining Pending Transactions: %d\n", pool.Count())

	return nil
}

// runMempoolPersistenceTest creates one fully valid pending transaction
// and intentionally leaves it unmined.
//
// Flow:
//  1. create miner/sender + receiver wallets
//  2. mine one empty funding block to sender
//  3. create and sign a 10 SUDH transaction
//  4. validate and add it to the mempool
//  5. save chain, state and mempool
//
// Restart the same node without -mempooltest to prove the transaction
// is restored from sudharma-mempool.json.
func runMempoolPersistenceTest(
	chain *blockchain.Chain,
	nodeState *blockchain.State,
	networkNode *p2p.Node,
	chainPath string,
	statePath string,
	mempoolPath string,
) error {

	if chain == nil {
		return fmt.Errorf(
			"chain cannot be nil",
		)
	}

	if nodeState == nil {
		return fmt.Errorf(
			"state cannot be nil",
		)
	}

	if networkNode == nil {
		return fmt.Errorf(
			"network node cannot be nil",
		)
	}

	if networkNode.MempoolCount() != 0 {
		return fmt.Errorf(
			"mempool persistence test requires an empty mempool",
		)
	}

	sender, err :=
		wallet.NewWallet()

	if err != nil {
		return fmt.Errorf(
			"failed creating sender wallet: %w",
			err,
		)
	}

	receiver, err :=
		wallet.NewWallet()

	if err != nil {
		return fmt.Errorf(
			"failed creating receiver wallet: %w",
			err,
		)
	}

	fmt.Println()
	fmt.Println(
		"========== SUDHARMA NETWORK MEMPOOL PERSISTENCE TEST ==========",
	)

	fmt.Printf(
		"Funding sender: %s\n",
		sender.Address,
	)

	fmt.Printf(
		"Receiver:       %s\n",
		receiver.Address,
	)

	fmt.Printf(
		"Mining funding block #%d...\n",
		chain.Height()+1,
	)

	result,
		reward,
		err :=
		miner.MineNextBlock(
			chain,
			nodeState,
			networkNode.Mempool(),
			sender.Address,
			1_000_000,
		)

	if err != nil {
		return fmt.Errorf(
			"failed mining funding block: %w",
			err,
		)
	}

	if !result.Found {
		return fmt.Errorf(
			"funding block was not found",
		)
	}

	networkNode.RefreshChainStatus()

	fmt.Printf(
		"Funding block found | Height: %d | Reward: %.8f SUDH\n",
		chain.Height(),
		float64(reward)/float64(params.CoinDecimals),
	)

	amount :=
		uint64(10) *
			params.CoinDecimals

	nonce, err :=
		nodeState.ExpectedNonce(
			sender.Address,
		)

	if err != nil {
		return fmt.Errorf(
			"failed determining nonce: %w",
			err,
		)
	}

	tx :=
		transactions.NewTransaction(
			sender.Address,
			receiver.Address,
			amount,
			nonce,
		)

	if err :=
		tx.Sign(
			sender,
		); err != nil {

		return fmt.Errorf(
			"failed signing transaction: %w",
			err,
		)
	}

	if err :=
		blockchain.ValidateMempoolTransaction(
			nodeState,
			networkNode.Mempool().AllTransactions(),
			tx,
		); err != nil {

		return fmt.Errorf(
			"pending transaction validation failed: %w",
			err,
		)
	}

	if err :=
		networkNode.Mempool().AddTransaction(
			tx,
		); err != nil {

		return fmt.Errorf(
			"failed adding pending transaction: %w",
			err,
		)
	}

	fmt.Println()
	fmt.Printf(
		"Pending Transaction ID: %s\n",
		tx.ID,
	)

	fmt.Printf(
		"From:                   %s\n",
		tx.From,
	)

	fmt.Printf(
		"To:                     %s\n",
		tx.To,
	)

	fmt.Println(
		"Amount:                 10.00000000 SUDH",
	)

	fmt.Printf(
		"Nonce:                  %d\n",
		tx.Nonce,
	)

	fmt.Printf(
		"Mempool before save:    %d transaction(s)\n",
		networkNode.MempoolCount(),
	)

	if err :=
		saveNodeData(
			chain,
			nodeState,
			chainPath,
			statePath,
		); err != nil {

		return err
	}

	if err :=
		networkNode.SaveMempoolToFile(
			mempoolPath,
		); err != nil {

		return fmt.Errorf(
			"failed saving mempool: %w",
			err,
		)
	}

	fmt.Printf(
		"Mempool saved to:       %s\n",
		mempoolPath,
	)

	fmt.Println(
		"Transaction intentionally left UNMINED.",
	)

	fmt.Println(
		"========================================================",
	)

	return nil
}
