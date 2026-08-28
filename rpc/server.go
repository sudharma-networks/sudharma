package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
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
	EnableMetrics   bool
}

func DefaultConfig() Config {
	return Config{ListenAddress: DefaultListenAddress, MaxBodyBytes: DefaultMaxBodyBytes, MaxConcurrent: DefaultMaxConcurrent, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, ShutdownTimeout: 5 * time.Second, EnableMetrics: true}
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
	s := &Server{config: config, node: node, chain: chain, state: state, limit: make(chan struct{}, config.MaxConcurrent)}
	s.server = &http.Server{Addr: config.ListenAddress, Handler: s.Handler(), ReadHeaderTimeout: config.ReadTimeout, ReadTimeout: config.ReadTimeout, WriteTimeout: config.WriteTimeout, IdleTimeout: config.IdleTimeout, MaxHeaderBytes: 32 << 10}
	return s, nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	if s.config.EnableMetrics {
		mux.HandleFunc("/metrics", s.handleMetrics)
	}
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/blocks/", s.handleBlock)
	mux.HandleFunc("/v1/accounts/", s.handleAccount)
	mux.HandleFunc("/v1/mempool", s.handleMempool)
	mux.HandleFunc("/v1/transactions", s.handleTransactions)
	mux.HandleFunc("/v1/transactions/", s.handleTransactionStatus)
	mux.HandleFunc("/", s.handleNotFound)
	return s.middleware(mux)
}
func (s *Server) ListenAndServe() error      { return s.server.ListenAndServe() }
func (s *Server) Serve(l net.Listener) error { return s.server.Serve(l) }
func (s *Server) Shutdown(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(c)
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
		defer func() {
			if recover() != nil {
				if !responseStarted(w) {
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}
		}()
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
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	height, tip := s.node.AdvertisedChainStatus()
	if tip == "" || s.chain.Tip() == nil {
		writeError(w, http.StatusServiceUnavailable, "node is not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "height": height, "tip_hash": tip})
}
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	height, _ := s.node.AdvertisedChainStatus()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "# HELP sudharma_chain_height Current advertised chain height.\n# TYPE sudharma_chain_height gauge\nsudharma_chain_height %d\n# HELP sudharma_peers Connected P2P peers.\n# TYPE sudharma_peers gauge\nsudharma_peers %d\n# HELP sudharma_mempool_transactions Transactions currently in the mempool.\n# TYPE sudharma_mempool_transactions gauge\nsudharma_mempool_transactions %d\n# HELP sudharma_issued_supply Native units issued by consensus.\n# TYPE sudharma_issued_supply gauge\nsudharma_issued_supply %d\n", height, s.node.PeerCount(), s.node.MempoolCount(), s.state.IssuedSupply())
}
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	height, tip := s.node.AdvertisedChainStatus()
	writeJSON(w, http.StatusOK, statusResponse{Network: "sudharma", Coin: params.CoinName, Symbol: params.CoinSymbol, NodeID: s.node.NodeID, P2PAddress: s.node.ListenAddress, Height: height, TipHash: tip, TotalWork: s.chain.TotalWork().String(), Peers: s.node.PeerCount(), Mempool: s.node.MempoolCount(), IssuedSupply: s.state.IssuedSupply()})
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
	h, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid block height")
		return
	}
	b, ok := s.chain.BlockByHeight(h)
	if !ok {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	writeJSON(w, http.StatusOK, b)
}
func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	a := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/accounts/"))
	if a == "" || strings.Contains(a, "/") || len(a) > 256 {
		writeError(w, http.StatusBadRequest, "invalid account address")
		return
	}
	n, err := s.state.ExpectedNonce(a)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, accountResponse{Address: a, Balance: s.state.Balance(a), ConfirmedNonce: s.state.AccountNonce(a), NextNonce: n})
}
func (s *Server) handleMempool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	limit := DefaultMempoolLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p < 1 || p > MaxMempoolLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return
		}
		limit = p
	}
	txs := s.node.Mempool().AllTransactions()
	sort.Slice(txs, func(i, j int) bool {
		if txs[i] == nil {
			return false
		}
		if txs[j] == nil {
			return true
		}
		return txs[i].ID < txs[j].ID
	})
	if len(txs) > limit {
		txs = txs[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": s.node.MempoolCount(), "transactions": txs})
}
func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxBodyBytes)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	var tx transactions.Transaction
	if err := d.Decode(&tx); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body exceeds maximum size")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid transaction JSON")
		return
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request must contain exactly one JSON object")
		return
	}
	relayed, err := s.node.SubmitTransaction(&tx)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, submitResponse{TransactionID: tx.ID, RelayedPeers: relayed, Accepted: true})
}
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "endpoint not found")
}
func methodNotAllowed(w http.ResponseWriter, a string) {
	w.Header().Set("Allow", a)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func responseStarted(w http.ResponseWriter) bool {
	type sr interface{ Written() bool }
	if r, ok := w.(sr); ok {
		return r.Written()
	}
	return false
}
