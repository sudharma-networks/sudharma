package p2p

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/sudharma-networks/sudharma/transactions"
)

const (
	ProtocolVersion uint32 = 1
	NetworkID              = "sudharma-testnet-1"
	// MainnetNetworkID is the frozen mainnet P2P namespace. Handshake still
	// uses NetworkID (public testnet) until mainnet launch is authorized.
	MainnetNetworkID = "sudharma-mainnet-1"

	MaxPeersPerMessage = 128

	// MaxHandshakeTotalWorkDigits bounds unauthenticated decimal big.Int input.
	// 128 decimal digits is far beyond any practical cumulative chain-work value
	// while preventing peers from forcing multi-megabyte integer parsing/storage.
	MaxHandshakeTotalWorkDigits = 128
)

type MessageType string

const (
	MessageHandshake   MessageType = "handshake"
	MessagePing        MessageType = "ping"
	MessagePong        MessageType = "pong"
	MessageTransaction MessageType = "transaction"
	MessageBlock       MessageType = "block"
	MessageGetBlocks   MessageType = "get_blocks"
	MessageBlocks      MessageType = "blocks"
	MessageGetMempool  MessageType = "get_mempool"
	MessageGetPeers    MessageType = "get_peers"
	MessagePeers       MessageType = "peers"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Handshake struct {
	ProtocolVersion uint32 `json:"protocol_version"`
	NetworkID       string `json:"network_id"`
	NodeID          string `json:"node_id"`
	ListenAddress   string `json:"listen_address"`
	Height          uint64 `json:"height"`
	TipHash         string `json:"tip_hash"`

	// TotalWork is encoded as a base-10 integer string.
	//
	// We intentionally do not use uint64 here because cumulative
	// chain work can eventually exceed 64-bit integer limits.
	TotalWork string `json:"total_work"`
}

type Ping struct {
	Nonce uint64 `json:"nonce"`
}

type Pong struct {
	Nonce uint64 `json:"nonce"`
}

type TransactionMessage struct {
	Transaction *transactions.Transaction `json:"transaction"`
}

// GetMempoolMessage requests the peer's current pending transactions.
// The payload is intentionally empty.
type GetMempoolMessage struct{}

// GetPeersMessage requests a peer discovery snapshot.
type GetPeersMessage struct{}

// PeersMessage carries a bounded list of known Sudharma Network peers.
type PeersMessage struct {
	Peers []KnownPeer `json:"peers"`
}

func encodeMessage(messageType MessageType, payload interface{}) ([]byte, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}
	message := Message{Type: messageType, Payload: payloadData}
	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	data = append(data, '\n')
	return data, nil
}

func NewHandshakeMessage(nodeID string, listenAddress string, height uint64, tipHash string, totalWork string) ([]byte, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}
	if totalWork == "" {
		return nil, fmt.Errorf("total work cannot be empty")
	}
	if len(totalWork) > MaxHandshakeTotalWorkDigits {
		return nil, fmt.Errorf("total work exceeds maximum encoded length")
	}
	work, ok := new(big.Int).SetString(totalWork, 10)
	if !ok {
		return nil, fmt.Errorf("total work is not a valid integer")
	}
	if work.Sign() < 0 {
		return nil, fmt.Errorf("total work cannot be negative")
	}
	return encodeMessage(MessageHandshake, Handshake{
		ProtocolVersion: ProtocolVersion,
		NetworkID:       LocalNetworkID(),
		NodeID:          nodeID,
		ListenAddress:   listenAddress,
		Height:          height,
		TipHash:         tipHash,
		TotalWork:       totalWork,
	})
}

func NewPingMessage(nonce uint64) ([]byte, error) {
	return encodeMessage(MessagePing, Ping{Nonce: nonce})
}

func NewPongMessage(nonce uint64) ([]byte, error) {
	return encodeMessage(MessagePong, Pong{Nonce: nonce})
}

// NewTransactionMessage creates a P2P transaction message.
func NewTransactionMessage(tx *transactions.Transaction) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	return encodeMessage(MessageTransaction, TransactionMessage{Transaction: tx})
}

// NewGetMempoolMessage requests a peer mempool snapshot.
// It should be sent only after blockchain synchronization is complete.
func NewGetMempoolMessage() ([]byte, error) {
	return encodeMessage(MessageGetMempool, GetMempoolMessage{})
}

// DecodeGetMempool validates an explicit mempool request.
func DecodeGetMempool(message *Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessageGetMempool {
		return fmt.Errorf("message is not a get_mempool request")
	}
	var request GetMempoolMessage
	if err := json.Unmarshal(message.Payload, &request); err != nil {
		return fmt.Errorf("invalid get_mempool message: %w", err)
	}
	return nil
}

// NewGetPeersMessage requests a peer discovery snapshot.
func NewGetPeersMessage() ([]byte, error) {
	return encodeMessage(MessageGetPeers, GetPeersMessage{})
}

// DecodeGetPeers validates a peer discovery request.
func DecodeGetPeers(message *Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessageGetPeers {
		return fmt.Errorf("message is not a get_peers request")
	}
	var request GetPeersMessage
	if err := json.Unmarshal(message.Payload, &request); err != nil {
		return fmt.Errorf("invalid get_peers message: %w", err)
	}
	return nil
}

// NewPeersMessage creates a bounded peer discovery response.
func NewPeersMessage(peers []KnownPeer) ([]byte, error) {
	if len(peers) > MaxPeersPerMessage {
		return nil, fmt.Errorf("too many peers in discovery message: %d > %d", len(peers), MaxPeersPerMessage)
	}
	normalized := make([]KnownPeer, 0, len(peers))
	seenNodeIDs := make(map[string]struct{})
	seenAddresses := make(map[string]struct{})
	for _, peer := range peers {
		if peer.NodeID == "" {
			return nil, fmt.Errorf("peer node ID cannot be empty")
		}
		if peer.Address == "" {
			return nil, fmt.Errorf("peer address cannot be empty")
		}
		if _, exists := seenNodeIDs[peer.NodeID]; exists {
			continue
		}
		if _, exists := seenAddresses[peer.Address]; exists {
			continue
		}
		seenNodeIDs[peer.NodeID] = struct{}{}
		seenAddresses[peer.Address] = struct{}{}
		normalized = append(normalized, peer)
	}
	return encodeMessage(MessagePeers, PeersMessage{Peers: normalized})
}

// DecodePeers validates and returns a peer discovery response.
func DecodePeers(message *Message) ([]KnownPeer, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessagePeers {
		return nil, fmt.Errorf("message is not a peers response")
	}
	var payload PeersMessage
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid peers message: %w", err)
	}
	if len(payload.Peers) > MaxPeersPerMessage {
		return nil, fmt.Errorf("too many peers in discovery message: %d > %d", len(payload.Peers), MaxPeersPerMessage)
	}
	normalized := make([]KnownPeer, 0, len(payload.Peers))
	seenNodeIDs := make(map[string]struct{})
	seenAddresses := make(map[string]struct{})
	for _, peer := range payload.Peers {
		if peer.NodeID == "" {
			return nil, fmt.Errorf("discovered peer node ID cannot be empty")
		}
		if peer.Address == "" {
			return nil, fmt.Errorf("discovered peer %s has empty address", peer.NodeID)
		}
		if _, exists := seenNodeIDs[peer.NodeID]; exists {
			continue
		}
		if _, exists := seenAddresses[peer.Address]; exists {
			continue
		}
		seenNodeIDs[peer.NodeID] = struct{}{}
		seenAddresses[peer.Address] = struct{}{}
		normalized = append(normalized, peer)
	}
	return normalized, nil
}

func DecodeMessage(data []byte) (*Message, error) {
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("invalid P2P message: %w", err)
	}
	switch message.Type {
	case MessageHandshake, MessagePing, MessagePong, MessageTransaction, MessageBlock, MessageGetBlocks, MessageBlocks, MessageGetMempool, MessageGetPeers, MessagePeers:
	default:
		return nil, fmt.Errorf("unknown P2P message type: %s", message.Type)
	}
	return &message, nil
}

func DecodeHandshake(message *Message) (*Handshake, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessageHandshake {
		return nil, fmt.Errorf("message is not a handshake")
	}
	var handshake Handshake
	if err := json.Unmarshal(message.Payload, &handshake); err != nil {
		return nil, fmt.Errorf("invalid handshake: %w", err)
	}
	if handshake.ProtocolVersion != ProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", handshake.ProtocolVersion)
	}
	if handshake.NetworkID != LocalNetworkID() {
		return nil, fmt.Errorf("wrong Sudharma Network network: %s", handshake.NetworkID)
	}
	if handshake.NodeID == "" {
		return nil, fmt.Errorf("remote node ID cannot be empty")
	}
	if handshake.TotalWork == "" {
		return nil, fmt.Errorf("remote total work cannot be empty")
	}
	if len(handshake.TotalWork) > MaxHandshakeTotalWorkDigits {
		return nil, fmt.Errorf("remote total work exceeds maximum encoded length")
	}
	work, ok := new(big.Int).SetString(handshake.TotalWork, 10)
	if !ok {
		return nil, fmt.Errorf("remote total work is not a valid integer")
	}
	if work.Sign() < 0 {
		return nil, fmt.Errorf("remote total work cannot be negative")
	}
	return &handshake, nil
}

func DecodePing(message *Message) (*Ping, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessagePing {
		return nil, fmt.Errorf("message is not ping")
	}
	var ping Ping
	if err := json.Unmarshal(message.Payload, &ping); err != nil {
		return nil, fmt.Errorf("invalid ping: %w", err)
	}
	return &ping, nil
}

func DecodePong(message *Message) (*Pong, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessagePong {
		return nil, fmt.Errorf("message is not pong")
	}
	var pong Pong
	if err := json.Unmarshal(message.Payload, &pong); err != nil {
		return nil, fmt.Errorf("invalid pong: %w", err)
	}
	return &pong, nil
}

// DecodeTransaction decodes and validates a transaction message.
func DecodeTransaction(message *Message) (*transactions.Transaction, error) {
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}
	if message.Type != MessageTransaction {
		return nil, fmt.Errorf("message is not a transaction")
	}
	var payload TransactionMessage
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid transaction message: %w", err)
	}
	if payload.Transaction == nil {
		return nil, fmt.Errorf("transaction payload is nil")
	}
	if !payload.Transaction.Verify() {
		return nil, fmt.Errorf("transaction signature verification failed")
	}
	return payload.Transaction, nil
}
