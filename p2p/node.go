package p2p

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/transactions"
)

const DefaultDialTimeout = 5 * time.Second

type PeerInfo struct {
	NodeID        string
	ListenAddress string
	Height        uint64
	TipHash       string
	TotalWork     string
}

type PeerConnection struct {
	Info PeerInfo

	conn          net.Conn
	reader        *bufio.Reader
	remoteAddress string

	writeMu sync.Mutex
}

type Node struct {
	mu sync.RWMutex

	NodeID        string
	ListenAddress string
	Height        uint64
	TipHash       string

	listener net.Listener
	peers    map[string]*PeerConnection

	discoveredPeers map[string]KnownPeer
	mempool         *mempool.Mempool
	state           *blockchain.State
	chain           *blockchain.Chain

	// Only one synchronous block-range request is allowed at a time for now.
	syncRequestMu  sync.Mutex
	blocksResponse chan []*blockchain.Block

	// Bounds unauthenticated inbound handshake work before peer admission.
	inboundHandshakeSlots chan struct{}
	// Tracks accepted inbound connections until handshake admission completes so
	// Stop can cancel unauthenticated work immediately instead of waiting for the
	// handshake deadline.
	inboundHandshakeConns map[net.Conn]struct{}
}

func NewNode(nodeID, listenAddress string, height uint64, tipHash string) (*Node, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}
	if listenAddress == "" {
		return nil, fmt.Errorf("listen address cannot be empty")
	}

	return &Node{
		NodeID:                 nodeID,
		ListenAddress:          listenAddress,
		Height:                 height,
		TipHash:                tipHash,
		peers:                  make(map[string]*PeerConnection),
		discoveredPeers:        make(map[string]KnownPeer),
		mempool:                mempool.NewMempool(),
		blocksResponse:         make(chan []*blockchain.Block, 1),
		inboundHandshakeSlots:  make(chan struct{}, MaxConcurrentInboundHandshakes),
		inboundHandshakeConns:  make(map[net.Conn]struct{}),
	}, nil
}

func (n *Node) SetState(state *blockchain.State) error {
	if state == nil {
		return fmt.Errorf("blockchain state cannot be nil")
	}
	n.mu.Lock()
	n.state = state
	n.mu.Unlock()
	return nil
}

func (n *Node) State() *blockchain.State {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

func (n *Node) localTotalWork() string {
	chain := n.Chain()
	if chain == nil {
		return "0"
	}
	return chain.TotalWork().String()
}

func (n *Node) Start() error {
	n.mu.Lock()
	if n.listener != nil {
		n.mu.Unlock()
		return fmt.Errorf("node is already running")
	}

	listener, err := net.Listen("tcp", n.ListenAddress)
	if err != nil {
		n.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", n.ListenAddress, err)
	}
	n.listener = listener
	n.ListenAddress = listener.Addr().String()
	n.mu.Unlock()

	go n.acceptLoop()
	return nil
}

func (n *Node) Stop() error {
	n.mu.Lock()
	listener := n.listener
	n.listener = nil
	peers := make([]*PeerConnection, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, peer)
	}
	n.peers = make(map[string]*PeerConnection)
	handshakes := make([]net.Conn, 0, len(n.inboundHandshakeConns))
	for conn := range n.inboundHandshakeConns {
		handshakes = append(handshakes, conn)
	}
	n.inboundHandshakeConns = make(map[net.Conn]struct{})
	n.mu.Unlock()

	for _, conn := range handshakes {
		if conn != nil {
			_ = conn.Close()
		}
	}
	for _, peer := range peers {
		if peer != nil && peer.conn != nil {
			_ = peer.conn.Close()
		}
	}
	if listener != nil {
		return listener.Close()
	}
	return nil
}

func (n *Node) acceptLoop() {
	for {
		n.mu.RLock()
		listener := n.listener
		n.mu.RUnlock()
		if listener == nil {
			return
		}

		conn, err := listener.Accept()
		if err != nil {
			// A listener close during Stop is terminal. Other accept failures are
			// treated as transient so one network hiccup does not kill the node's
			// inbound connectivity loop.
			n.mu.RLock()
			stillRunning := n.listener == listener
			n.mu.RUnlock()
			if !stillRunning {
				return
			}
			time.Sleep(25 * time.Millisecond)
			continue
		}

		if !n.tryAcquireInboundHandshake() {
			_ = conn.Close()
			continue
		}
		if !n.trackInboundHandshake(listener, conn) {
			n.releaseInboundHandshake()
			_ = conn.Close()
			continue
		}
		go func() {
			defer n.releaseInboundHandshake()
			defer n.untrackInboundHandshake(conn)
			n.handleIncomingConnection(conn)
		}()
	}
}

func (n *Node) handleIncomingConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	_ = conn.SetDeadline(time.Now().Add(DefaultDialTimeout))

	data, err := readBoundedPeerMessage(reader)
	if err != nil {
		_ = conn.Close()
		return
	}
	message, err := DecodeMessage(data)
	if err != nil {
		_ = conn.Close()
		return
	}
	handshake, err := DecodeHandshake(message)
	if err != nil {
		_ = conn.Close()
		return
	}

	response, err := NewHandshakeMessage(n.NodeID, n.ListenAddress, n.Height, n.TipHash, n.localTotalWork())
	if err != nil {
		_ = conn.Close()
		return
	}
	if _, err := conn.Write(response); err != nil {
		_ = conn.Close()
		return
	}
	_ = conn.SetDeadline(time.Time{})

	peer := &PeerConnection{
		Info: PeerInfo{
			NodeID:        handshake.NodeID,
			ListenAddress: handshake.ListenAddress,
			Height:        handshake.Height,
			TipHash:       handshake.TipHash,
			TotalWork:     handshake.TotalWork,
		},
		conn:          conn,
		reader:        reader,
		remoteAddress: conn.RemoteAddr().String(),
	}
	if !n.storePeer(peer) {
		_ = conn.Close()
		return
	}
	go n.readLoop(peer)
}

func (n *Node) Connect(address string) (*PeerInfo, error) {
	if address == "" {
		return nil, fmt.Errorf("peer address cannot be empty")
	}

	conn, err := net.DialTimeout("tcp", address, DefaultDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %w", address, err)
	}
	reader := bufio.NewReader(conn)
	_ = conn.SetDeadline(time.Now().Add(DefaultDialTimeout))

	handshakeData, err := NewHandshakeMessage(n.NodeID, n.ListenAddress, n.Height, n.TipHash, n.localTotalWork())
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.Write(handshakeData); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	data, err := readBoundedPeerMessage(reader)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read handshake response: %w", err)
	}
	message, err := DecodeMessage(data)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	handshake, err := DecodeHandshake(message)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})

	peer := &PeerConnection{
		Info: PeerInfo{
			NodeID:        handshake.NodeID,
			ListenAddress: handshake.ListenAddress,
			Height:        handshake.Height,
			TipHash:       handshake.TipHash,
			TotalWork:     handshake.TotalWork,
		},
		conn:          conn,
		reader:        reader,
		remoteAddress: conn.RemoteAddr().String(),
	}
	if !n.storePeer(peer) {
		_ = conn.Close()
		return nil, fmt.Errorf("peer already connected or rejected by connection policy")
	}
	go n.readLoop(peer)
	info := peer.Info
	return &info, nil
}

func (n *Node) storePeer(peer *PeerConnection) bool {
	if peer == nil || peer.Info.NodeID == "" || peer.Info.NodeID == n.NodeID {
		return false
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.peers[peer.Info.NodeID]; exists {
		return false
	}
	if !n.canStorePeerLocked(peer) {
		return false
	}
	n.peers[peer.Info.NodeID] = peer
	return true
}

func (n *Node) removePeer(nodeID string) {
	n.mu.Lock()
	delete(n.peers, nodeID)
	n.mu.Unlock()
}

func (n *Node) punishPeer(peer *PeerConnection, amount int, reason string) bool {
	return n.penalizePeerAndMaybeDisconnect(peer, amount, reason) <= PeerDisconnectThreshold
}

func (n *Node) readLoop(peer *PeerConnection) {
	defer func() {
		n.removePeerConnection(peer)
		if peer.conn != nil {
			_ = peer.conn.Close()
		}
	}()

	for {
		if peer == nil || peer.conn == nil || peer.reader == nil {
			return
		}
		if err := setPeerReadDeadline(peer.conn); err != nil {
			return
		}
		data, err := readBoundedPeerMessage(peer.reader)
		if err != nil {
			return
		}

		message, err := DecodeMessage(data)
		if err != nil {
			if n.punishPeer(peer, PeerPenaltyProtocolAbuse, "malformed protocol message") {
				return
			}
			continue
		}

		switch message.Type {
		case MessagePing:
			ping, err := DecodePing(message)
			if err != nil {
				if n.punishPeer(peer, PeerPenaltyMalformed, "malformed ping") {
					return
				}
				continue
			}
			pong, err := NewPongMessage(ping.Nonce)
			if err != nil {
				continue
			}
			_ = peer.write(pong)

		case MessagePong:
			// Reserved for future latency tracking. Bare pongs are not rewarded,
			// preventing reputation farming through cheap keepalive spam.

		case MessageTransaction:
			tx, err := DecodeTransaction(message)
			if err != nil {
				fmt.Printf("[TX] Rejected malformed transaction from %s: %v\n", peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyMalformed, "malformed transaction") {
					return
				}
				continue
			}
			if _, exists := n.mempool.GetTransaction(tx.ID); exists {
				fmt.Printf("[TX] Duplicate transaction ignored: %s\n", tx.ID)
				continue
			}
			state := n.State()
			if state == nil {
				fmt.Printf("[TX] Rejected %s: blockchain state unavailable\n", tx.ID)
				continue
			}
			pending := n.mempool.AllTransactions()
			if err := blockchain.ValidateMempoolTransaction(state, pending, tx); err != nil {
				fmt.Printf("[TX] Rejected %s from %s: %v\n", tx.ID, peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyInvalidData, "invalid transaction") {
					return
				}
				continue
			}
			if err := n.mempool.AddTransaction(tx); err != nil {
				fmt.Printf("[TX] Mempool add failed for %s: %v\n", tx.ID, err)
				continue
			}
			n.rewardValidPeerMessage(peer)
			fmt.Printf("[TX] Accepted %s from %s | Mempool: %d\n", tx.ID, peer.Info.NodeID, n.mempool.Count())
			if sent, err := n.relayTransaction(tx, peer.Info.NodeID); err != nil {
				fmt.Printf("[TX] Gossip failed for %s: %v\n", tx.ID, err)
			} else if sent > 0 {
				fmt.Printf("[TX] Relayed %s to %d peer(s)\n", tx.ID, sent)
			}

		case MessageGetMempool:
			if err := DecodeGetMempool(message); err != nil {
				fmt.Printf("[MEMPOOL] Invalid mempool request from %s: %v\n", peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyMalformed, "malformed mempool request") {
					return
				}
				continue
			}
			sent, err := n.syncMempoolToPeer(peer)
			if err != nil {
				fmt.Printf("[MEMPOOL] Failed syncing mempool to %s: %v\n", peer.Info.NodeID, err)
				continue
			}
			n.rewardValidPeerMessage(peer)
			fmt.Printf("[MEMPOOL] Responded to %s with %d pending transaction(s)\n", peer.Info.NodeID, sent)

		case MessageGetPeers:
			if err := DecodeGetPeers(message); err != nil {
				fmt.Printf("[PEERS] Invalid peer discovery request from %s: %v\n", peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyMalformed, "malformed peer discovery request") {
					return
				}
				continue
			}
			sent, err := n.sendPeersToPeer(peer)
			if err != nil {
				fmt.Printf("[PEERS] Failed responding to %s: %v\n", peer.Info.NodeID, err)
				continue
			}
			n.rewardValidPeerMessage(peer)
			fmt.Printf("[PEERS] Responded to %s with %d peer(s)\n", peer.Info.NodeID, sent)

		case MessagePeers:
			discovered, err := DecodePeers(message)
			if err != nil {
				fmt.Printf("[PEERS] Invalid peer discovery response from %s: %v\n", peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyMalformed, "malformed peer discovery response") {
					return
				}
				continue
			}
			added := n.mergeDiscoveredPeers(discovered)
			n.rewardValidPeerMessage(peer)
			fmt.Printf("[PEERS] Learned %d new peer(s) from %s\n", added, peer.Info.NodeID)
			if added > 0 {
				connected, failed := n.AutoConnectDiscoveredPeers()
				fmt.Printf("[PEERS] Discovery auto-connect complete | Connected: %d | Failed: %d\n", connected, failed)
			}

		case MessageBlock:
			block, err := DecodeBlock(message)
			if err != nil {
				fmt.Printf("[BLOCK] Rejected malformed block from %s: %v\n", peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyMalformed, "malformed block") {
					return
				}
				continue
			}
			if err := n.AcceptBlock(block); err != nil {
				fmt.Printf("[BLOCK] Rejected block from %s: %v\n", peer.Info.NodeID, err)
				if n.punishPeer(peer, PeerPenaltyInvalidData, "invalid block") {
					return
				}
				continue
			}
			n.rewardValidPeerMessage(peer)
			fmt.Printf("[BLOCK] Accepted block #%d from %s | Tip: %s\n", block.Height, peer.Info.NodeID, block.Hash())
			if sent, err := n.relayBlock(block, peer.Info.NodeID); err != nil {
				fmt.Printf("[BLOCK] Gossip failed for block #%d: %v\n", block.Height, err)
			} else if sent > 0 {
				fmt.Printf("[BLOCK] Relayed block #%d to %d peer(s)\n", block.Height, sent)
			}

		case MessageGetBlocks:
			n.handleGetBlocksSecure(peer, message)

		case MessageBlocks:
			n.handleBlocksResponseSecure(peer, message)

		default:
			fmt.Printf("[PEERS] Unknown message type %q from %s\n", message.Type, peer.Info.NodeID)
			if n.punishPeer(peer, PeerPenaltyProtocolAbuse, "unknown protocol message type") {
				return
			}
		}
	}
}

func (p *PeerConnection) write(data []byte) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("peer connection is unavailable")
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	if err := setPeerWriteDeadline(p.conn); err != nil {
		return err
	}
	written := 0
	for written < len(data) {
		n, err := p.conn.Write(data[written:])
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrUnexpectedEOF
		}
		written += n
	}
	return nil
}

func (n *Node) SendPing(nodeID string, nonce uint64) error {
	n.mu.RLock()
	peer, ok := n.peers[nodeID]
	n.mu.RUnlock()
	if !ok {
		return fmt.Errorf("peer not found: %s", nodeID)
	}
	data, err := NewPingMessage(nonce)
	if err != nil {
		return err
	}
	if err := peer.write(data); err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}
	return nil
}

func (n *Node) BroadcastTransaction(tx *transactions.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	if !tx.Verify() {
		return fmt.Errorf("cannot broadcast invalid transaction")
	}
	data, err := NewTransactionMessage(tx)
	if err != nil {
		return err
	}

	n.mu.RLock()
	peers := make([]*PeerConnection, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, peer)
	}
	n.mu.RUnlock()

	for _, peer := range peers {
		if err := peer.write(data); err != nil {
			return fmt.Errorf("failed to broadcast transaction to %s: %w", peer.Info.NodeID, err)
		}
	}
	return nil
}

func (n *Node) MempoolCount() int {
	return n.mempool.Count()
}

func (n *Node) MempoolTransaction(txID string) (*transactions.Transaction, bool) {
	return n.mempool.GetTransaction(txID)
}

func (n *Node) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.peers)
}

func (n *Node) Peer(nodeID string) (PeerInfo, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	peer, ok := n.peers[nodeID]
	if !ok {
		return PeerInfo{}, false
	}
	return peer.Info, true
}
