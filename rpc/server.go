package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/p2p"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

const (
	DefaultListenAddress = "127.0.0.1:18545"
	DefaultMaxBodyBytes  = int64(1 << 20)
	DefaultMaxConcurrent = 128
	DefaultMempoolLimit  = 100
	MaxMempoolLimit      = 500
)

type Config struct {
	ListenAddress   string
	MaxBodyBytes    int64
	MaxConcurrent   int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		ListenAddress:   DefaultListenAddress,
		MaxBodyBytes:    DefaultMaxBodyBytes,
		MaxConcurrent:   DefaultMaxConcurrent,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

type Server struct {
	config Config
	node   *p2p.Node
	chain  *blockchain.Chain
	state  *blockchain.State
	server *http.Server
	limit  chan struct{}
}

type errorResponse struct {
	Error string `json:"error"`
}

type statusResponse struct {
	Network      string `json:"network"`
	Coin         string `json:"coin"`
	Symbol       string `json:"symbol"`
	NodeID       string `json:"node_id"`
	P2PAddress   string `json:"p2p_address"`
	Height       uint64 `json:"height"`
	TipHash      string `json:"tip_hash"`
	TotalWork    string `json:"total_work"`
	Peers        int    `json:"peers"`
	Mempool      int    `json:"mempool"`
	IssuedSupply uint64 `json:"issued_supply"`
}

type accountResponse struct {
	Address        string `json:"address"`
	Balance        uint64 `json:"balance"`
	ConfirmedNonce uint64 `json:"confirmed_nonce"`
	NextNonce      uint64 `json:"next_nonce"`
}

type submitResponse struct {
	TransactionID string `json:"transaction_id"`
	RelayedPeers  int    `json:"relayed_peers"`
	Accepted      bool   `json:"accepted"`
}

func NewServer(config Config, node *p2p.Node, chain *blockchain.Chain, state *blockchain.State) (*Server, error) {
	if node == nil || chain == nil || state == nil {
		return nil, fmt.Errorf("rpc dependencies cannot be nil")
	}
	if config.ListenAddress == "" {
		config.ListenAddress = DefaultListenAddress
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = DefaultMaxBodyBytes
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = DefaultMaxConcurrent
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 5 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 10 * time.Second
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 30 * time.Second
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 5 * time.Second
	}

	s := &Server{
		config: config,
		node:   node,
		chain:  chain,
		state:  state,
		limit:  make(chan struct{}, config.MaxConcurrent),
	}
	s.server = &http.Server{
		Addr:              config.ListenAddress,
		Handler:           s.Handler(),
		ReadHeaderTimeout: config.ReadTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    32 << 10,
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/blocks/", s.handleBlock)
	mux.HandleFunc("/v1/accounts/", s.handleAccount)
	mux.HandleFunc("/v1/mempool", s.handleMempool)
	mux.HandleFunc("/v1/transactions", s.handleTransactions)
	return s.middleware(mux)
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Serve(listener net.Listener) error {
	return s.server.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(shutdownCtx)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		select {
		case s.limit <- struct{}{}:
			defer func() { <-s.limit }()
		default:
			writeError(w, http.StatusServiceUnavailable, "rpc server is busy")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	height, tip := s.node.AdvertisedChainStatus()
	writeJSON(w, http.StatusOK, statusResponse{
		Network:      "sudharma",
		Coin:         params.CoinName,
		Symbol:       params.CoinSymbol,
		NodeID:       s.node.NodeID,
		P2PAddress:   s.node.ListenAddress,
		Height:       height,
		TipHash:      tip,
		TotalWork:    s.chain.TotalWork().String(),
		Peers:        s.node.PeerCount(),
		Mempool:      s.node.MempoolCount(),
		IssuedSupply: s.state.IssuedSupply(),
	})
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	raw := strings.TrimPrefix(r.URL.Path, "/v1/blocks/")
	if raw == "" || strings.Contains(raw, "/") {
		writeError(w, http.StatusBadRequest, "block height is required")
		return
	}
	height, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid block height")
		return
	}
	block, ok := s.chain.BlockByHeight(height)
	if !ok {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	writeJSON(w, http.StatusOK, block)
}

func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	address := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/accounts/"))
	if address == "" || strings.Contains(address, "/") || len(address) > 256 {
		writeError(w, http.StatusBadRequest, "invalid account address")
		return
	}
	nextNonce, err := s.state.ExpectedNonce(address)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, accountResponse{
		Address:        address,
		Balance:        s.state.Balance(address),
		ConfirmedNonce: s.state.AccountNonce(address),
		NextNonce:      nextNonce,
	})
}

func (s *Server) handleMempool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	limit := DefaultMempoolLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > MaxMempoolLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return
		}
		limit = parsed
	}
	txs := s.node.Mempool().AllTransactions()
	if len(txs) > limit {
		txs = txs[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":        s.node.MempoolCount(),
		"transactions": txs,
	})
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if contentType := r.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var tx transactions.Transaction
	if err := decoder.Decode(&tx); err != nil {
		writeError(w, http.StatusBadRequest, "invalid transaction JSON")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request must contain exactly one JSON object")
		return
	}

	relayed, err := s.node.SubmitTransaction(&tx)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, submitResponse{
		TransactionID: tx.ID,
		RelayedPeers:  relayed,
		Accepted:      true,
	})
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
