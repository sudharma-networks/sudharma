package p2p

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/transactions"
)

const (
	DefaultDialTimeout = 5 * time.Second
)

type PeerInfo struct {
	NodeID        string
	ListenAddress string
	Height        uint64
	TipHash       string
	TotalWork     string
}

type PeerConnection struct {
	Info PeerInfo

	conn   net.Conn
	reader *bufio.Reader

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

	// Only one synchronous block-range request
	// is allowed at a time for now.
	syncRequestMu sync.Mutex

	blocksResponse chan []*blockchain.Block
}

func NewNode(
	nodeID string,
	listenAddress string,
	height uint64,
	tipHash string,
) (*Node, error) {

	if nodeID == "" {
		return nil, fmt.Errorf(
			"node ID cannot be empty",
		)
	}

	if listenAddress == "" {
		return nil, fmt.Errorf(
			"listen address cannot be empty",
		)
	}

	return &Node{
		NodeID:          nodeID,
		ListenAddress:   listenAddress,
		Height:          height,
		TipHash:         tipHash,
		peers:           make(map[string]*PeerConnection),
		discoveredPeers: make(map[string]KnownPeer),
		mempool:         mempool.NewMempool(),
		blocksResponse:  make(chan []*blockchain.Block, 1),
	}, nil
}

// SetState attaches the current Sudharma Network blockchain state
// used for validating transactions before mempool admission.
func (n *Node) SetState(
	state *blockchain.State,
) error {

	if state == nil {
		return fmt.Errorf(
			"blockchain state cannot be nil",
		)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.state = state

	return nil
}

// State returns the currently attached blockchain state.
func (n *Node) State() *blockchain.State {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.state
}

// localTotalWork returns the cumulative Proof-of-Work
// of the currently attached chain as a base-10 string.
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

		return fmt.Errorf(
			"node is already running",
		)
	}

	listener, err := net.Listen(
		"tcp",
		n.ListenAddress,
	)

	if err != nil {
		n.mu.Unlock()

		return fmt.Errorf(
			"failed to listen on %s: %w",
			n.ListenAddress,
			err,
		)
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

	peers := make(
		[]*PeerConnection,
		0,
		len(n.peers),
	)

	for _, peer := range n.peers {
		peers = append(
			peers,
			peer,
		)
	}

	n.peers = make(
		map[string]*PeerConnection,
	)

	n.mu.Unlock()

	for _, peer := range peers {
		_ = peer.conn.Close()
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
			return
		}

		go n.handleIncomingConnection(
			conn,
		)
	}
}

func (n *Node) handleIncomingConnection(
	conn net.Conn,
) {
	reader := bufio.NewReader(conn)

	_ = conn.SetDeadline(
		time.Now().Add(DefaultDialTimeout),
	)

	data, err := reader.ReadBytes('\n')
	if err != nil {
		_ = conn.Close()
		return
	}

	message, err := DecodeMessage(data)
	if err != nil {
		_ = conn.Close()
		return
	}

	handshake, err := DecodeHandshake(
		message,
	)

	if err != nil {
		_ = conn.Close()
		return
	}

	response, err := NewHandshakeMessage(
		n.NodeID,
		n.ListenAddress,
		n.Height,
		n.TipHash,
		n.localTotalWork(),
	)

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
		conn:   conn,
		reader: reader,
	}

	if !n.storePeer(peer) {
		_ = conn.Close()
		return
	}

	go n.readLoop(peer)
}

func (n *Node) Connect(
	address string,
) (*PeerInfo, error) {

	if address == "" {
		return nil, fmt.Errorf(
			"peer address cannot be empty",
		)
	}

	conn, err := net.DialTimeout(
		"tcp",
		address,
		DefaultDialTimeout,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to connect to peer %s: %w",
			address,
			err,
		)
	}

	reader := bufio.NewReader(conn)

	_ = conn.SetDeadline(
		time.Now().Add(DefaultDialTimeout),
	)

	handshakeData, err :=
		NewHandshakeMessage(
			n.NodeID,
			n.ListenAddress,
			n.Height,
			n.TipHash,
			n.localTotalWork(),
		)

	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if _, err := conn.Write(
		handshakeData,
	); err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf(
			"failed to send handshake: %w",
			err,
		)
	}

	data, err := reader.ReadBytes('\n')

	if err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf(
			"failed to read handshake response: %w",
			err,
		)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	handshake, err :=
		DecodeHandshake(message)

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
		conn:   conn,
		reader: reader,
	}

	if !n.storePeer(peer) {
		_ = conn.Close()

		return nil, fmt.Errorf(
			"peer already connected or invalid",
		)
	}

	go n.readLoop(peer)

	info := peer.Info

	return &info, nil
}

func (n *Node) storePeer(
	peer *PeerConnection,
) bool {

	if peer == nil {
		return false
	}

	if peer.Info.NodeID == "" {
		return false
	}

	if peer.Info.NodeID == n.NodeID {
		return false
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists :=
		n.peers[peer.Info.NodeID]; exists {

		return false
	}

	n.peers[peer.Info.NodeID] = peer

	return true
}

func (n *Node) removePeer(
	nodeID string,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(
		n.peers,
		nodeID,
	)
}

func (n *Node) readLoop(
	peer *PeerConnection,
) {
	defer func() {
		n.removePeer(
			peer.Info.NodeID,
		)

		_ = peer.conn.Close()
	}()

	for {
		data, err :=
			peer.reader.ReadBytes('\n')

		if err != nil {
			return
		}

		message, err :=
			DecodeMessage(data)

		if err != nil {
			continue
		}

		switch message.Type {

		// ----------------------------------------
		// Ping
		// ----------------------------------------

		case MessagePing:
			ping, err :=
				DecodePing(message)

			if err != nil {
				continue
			}

			pong, err :=
				NewPongMessage(
					ping.Nonce,
				)

			if err != nil {
				continue
			}

			_ = peer.write(pong)

		// ----------------------------------------
		// Pong
		// ----------------------------------------

		case MessagePong:
			// Reserved for future latency tracking.

		// ----------------------------------------
		// Transaction
		// ----------------------------------------

		case MessageTransaction:
			tx, err :=
				DecodeTransaction(message)

			if err != nil {
				fmt.Printf(
					"[TX] Rejected malformed transaction from %s: %v\n",
					peer.Info.NodeID,
					err,
				)

				continue
			}

			if _, exists :=
				n.mempool.GetTransaction(
					tx.ID,
				); exists {

				fmt.Printf(
					"[TX] Duplicate transaction ignored: %s\n",
					tx.ID,
				)

				continue
			}

			state :=
				n.State()

			if state == nil {
				fmt.Printf(
					"[TX] Rejected %s: blockchain state unavailable\n",
					tx.ID,
				)

				continue
			}

			pending :=
				n.mempool.AllTransactions()

			if err :=
				blockchain.ValidateMempoolTransaction(
					state,
					pending,
					tx,
				); err != nil {

				fmt.Printf(
					"[TX] Rejected %s from %s: %v\n",
					tx.ID,
					peer.Info.NodeID,
					err,
				)

				continue
			}

			if err :=
				n.mempool.AddTransaction(
					tx,
				); err != nil {

				fmt.Printf(
					"[TX] Mempool add failed for %s: %v\n",
					tx.ID,
					err,
				)

				continue
			}

			fmt.Printf(
				"[TX] Accepted %s from %s | Mempool: %d\n",
				tx.ID,
				peer.Info.NodeID,
				n.mempool.Count(),
			)

			if sent, err :=
				n.relayTransaction(
					tx,
					peer.Info.NodeID,
				); err != nil {

				fmt.Printf(
					"[TX] Gossip failed for %s: %v\n",
					tx.ID,
					err,
				)

			} else if sent > 0 {
				fmt.Printf(
					"[TX] Relayed %s to %d peer(s)\n",
					tx.ID,
					sent,
				)
			}

		// ----------------------------------------
		// Explicit Mempool Sync Request
		// ----------------------------------------

		case MessageGetMempool:
			if err := DecodeGetMempool(message); err != nil {
				fmt.Printf(
					"[MEMPOOL] Invalid mempool request from %s: %v\n",
					peer.Info.NodeID,
					err,
				)
				continue
			}

			sent, err :=
				n.syncMempoolToPeer(
					peer,
				)

			if err != nil {
				fmt.Printf(
					"[MEMPOOL] Failed syncing mempool to %s: %v\n",
					peer.Info.NodeID,
					err,
				)
				continue
			}

			fmt.Printf(
				"[MEMPOOL] Responded to %s with %d pending transaction(s)\n",
				peer.Info.NodeID,
				sent,
			)
		// ----------------------------------------
		// Peer Discovery Request
		// ----------------------------------------

		case MessageGetPeers:
			if err := DecodeGetPeers(message); err != nil {
				fmt.Printf(
					"[PEERS] Invalid peer discovery request from %s: %v\n",
					peer.Info.NodeID,
					err,
				)

				continue
			}

			sent, err :=
				n.sendPeersToPeer(
					peer,
				)

			if err != nil {
				fmt.Printf(
					"[PEERS] Failed responding to %s: %v\n",
					peer.Info.NodeID,
					err,
				)

				continue
			}

			fmt.Printf(
				"[PEERS] Responded to %s with %d peer(s)\n",
				peer.Info.NodeID,
				sent,
			)

		// ----------------------------------------
		// Peer Discovery Response
		// ----------------------------------------

		case MessagePeers:
			discovered, err :=
				DecodePeers(
					message,
				)

			if err != nil {
				fmt.Printf(
					"[PEERS] Invalid peer discovery response from %s: %v\n",
					peer.Info.NodeID,
					err,
				)

				continue
			}

			added :=
				n.mergeDiscoveredPeers(
					discovered,
				)

			fmt.Printf(
				"[PEERS] Learned %d new peer(s) from %s\n",
				added,
				peer.Info.NodeID,
			)

			if added > 0 {
				connected,
					failed :=
					n.AutoConnectDiscoveredPeers()

				fmt.Printf(
					"[PEERS] Discovery auto-connect complete | Connected: %d | Failed: %d\n",
					connected,
					failed,
				)
			}

		// ----------------------------------------
		// Single Block
		// ----------------------------------------

		case MessageBlock:
			block, err :=
				DecodeBlock(message)

			if err != nil {
				fmt.Printf(
					"[BLOCK] Rejected malformed block from %s: %v\n",
					peer.Info.NodeID,
					err,
				)

				continue
			}

			if err :=
				n.AcceptBlock(
					block,
				); err != nil {

				fmt.Printf(
					"[BLOCK] Rejected block from %s: %v\n",
					peer.Info.NodeID,
					err,
				)

				continue
			}

			fmt.Printf(
				"[BLOCK] Accepted block #%d from %s | Tip: %s\n",
				block.Height,
				peer.Info.NodeID,
				block.Hash(),
			)

			if sent, err :=
				n.relayBlock(
					block,
					peer.Info.NodeID,
				); err != nil {

				fmt.Printf(
					"[BLOCK] Gossip failed for block #%d: %v\n",
					block.Height,
					err,
				)

			} else if sent > 0 {
				fmt.Printf(
					"[BLOCK] Relayed block #%d to %d peer(s)\n",
					block.Height,
					sent,
				)
			}

		// ----------------------------------------
		// Chain Sync Request
		// ----------------------------------------

		case MessageGetBlocks:
			n.handleGetBlocks(
				peer,
				message,
			)

		// ----------------------------------------
		// Chain Sync Response
		// ----------------------------------------

		case MessageBlocks:
			n.handleBlocksResponse(
				peer,
				message,
			)
		}
	}
}

func (p *PeerConnection) write(
	data []byte,
) error {

	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	_, err := p.conn.Write(data)

	return err
}

func (n *Node) SendPing(
	nodeID string,
	nonce uint64,
) error {

	n.mu.RLock()

	peer, ok :=
		n.peers[nodeID]

	n.mu.RUnlock()

	if !ok {
		return fmt.Errorf(
			"peer not found: %s",
			nodeID,
		)
	}

	data, err :=
		NewPingMessage(nonce)

	if err != nil {
		return err
	}

	if err := peer.write(data); err != nil {
		return fmt.Errorf(
			"failed to send ping: %w",
			err,
		)
	}

	return nil
}

func (n *Node) BroadcastTransaction(
	tx *transactions.Transaction,
) error {

	if tx == nil {
		return fmt.Errorf(
			"transaction cannot be nil",
		)
	}

	if !tx.Verify() {
		return fmt.Errorf(
			"cannot broadcast invalid transaction",
		)
	}

	data, err :=
		NewTransactionMessage(tx)

	if err != nil {
		return err
	}

	n.mu.RLock()

	peers := make(
		[]*PeerConnection,
		0,
		len(n.peers),
	)

	for _, peer := range n.peers {
		peers = append(
			peers,
			peer,
		)
	}

	n.mu.RUnlock()

	for _, peer := range peers {
		if err := peer.write(data); err != nil {
			return fmt.Errorf(
				"failed to broadcast transaction to %s: %w",
				peer.Info.NodeID,
				err,
			)
		}
	}

	return nil
}

func (n *Node) MempoolCount() int {
	return n.mempool.Count()
}

func (n *Node) MempoolTransaction(
	txID string,
) (*transactions.Transaction, bool) {

	return n.mempool.GetTransaction(txID)
}

func (n *Node) PeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return len(n.peers)
}

func (n *Node) Peer(
	nodeID string,
) (PeerInfo, bool) {

	n.mu.RLock()
	defer n.mu.RUnlock()

	peer, ok :=
		n.peers[nodeID]

	if !ok {
		return PeerInfo{}, false
	}

	return peer.Info, true
}
