package rpc

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

const (
	DefaultExplorerLimit = 20
	MaxExplorerLimit     = 100
)

type explorerStatusResponse struct {
	Network      string `json:"network"`
	Coin         string `json:"coin"`
	Symbol       string `json:"symbol"`
	Height       uint64 `json:"height"`
	TipHash      string `json:"tip_hash"`
	TotalWork    string `json:"total_work"`
	Peers        int    `json:"peers"`
	Mempool      int    `json:"mempool"`
	IssuedSupply uint64 `json:"issued_supply"`
}

type explorerTransactionView struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
	Fee    uint64 `json:"fee"`
	Nonce  uint64 `json:"nonce"`
}

type explorerTransactionResponse struct {
	Transaction    explorerTransactionView `json:"transaction"`
	Status         string                  `json:"status"`
	BlockHeight    *uint64                 `json:"block_height,omitempty"`
	BlockHash      string                  `json:"block_hash,omitempty"`
	BlockTimestamp int64                   `json:"block_timestamp,omitempty"`
	Confirmations  uint64                  `json:"confirmations"`
}

type explorerBlockResponse struct {
	Height           uint64                    `json:"height"`
	Hash             string                    `json:"hash"`
	Timestamp        int64                     `json:"timestamp"`
	PreviousHash     string                    `json:"previous_hash"`
	MerkleRoot       string                    `json:"merkle_root"`
	Difficulty       uint32                    `json:"difficulty"`
	Nonce            uint64                    `json:"nonce"`
	MinerAddress     string                    `json:"miner_address"`
	TransactionCount int                       `json:"transaction_count"`
	Transactions     []explorerTransactionView `json:"transactions,omitempty"`
}

type explorerBlocksResponse struct {
	Blocks     []explorerBlockResponse `json:"blocks"`
	NextBefore *uint64                 `json:"next_before,omitempty"`
}

type explorerTransactionsResponse struct {
	Transactions     []explorerTransactionResponse `json:"transactions"`
	NextBeforeHeight *uint64                       `json:"next_before_height,omitempty"`
}

type explorerAddressResponse struct {
	Address          string                        `json:"address"`
	Balance          uint64                        `json:"balance"`
	ConfirmedNonce   uint64                        `json:"confirmed_nonce"`
	NextNonce        uint64                        `json:"next_nonce"`
	Transactions     []explorerTransactionResponse `json:"transactions"`
	NextBeforeHeight *uint64                       `json:"next_before_height,omitempty"`
}

type explorerSearchResponse struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

func (s *Server) handleExplorerStatus(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	tip := s.chain.Tip()
	if tip == nil {
		writeError(w, http.StatusServiceUnavailable, "chain is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, explorerStatusResponse{
		Network:      "sudharma",
		Coin:         params.CoinName,
		Symbol:       params.CoinSymbol,
		Height:       s.chain.Height(),
		TipHash:      tip.Hash(),
		TotalWork:    s.chain.TotalWork().String(),
		Peers:        s.node.PeerCount(),
		Mempool:      s.node.MempoolCount(),
		IssuedSupply: s.state.IssuedSupply(),
	})
}

func (s *Server) handleExplorerBlocks(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	limit, ok := explorerLimit(w, r)
	if !ok {
		return
	}
	before, ok := explorerOptionalHeight(w, r, "before")
	if !ok {
		return
	}
	blocks := s.chain.RecentBlocks(limit, before)
	views := make([]explorerBlockResponse, 0, len(blocks))
	for _, block := range blocks {
		views = append(views, explorerBlockView(block, false))
	}
	var next *uint64
	if len(blocks) == limit && len(blocks) > 0 {
		height := blocks[len(blocks)-1].Height
		next = &height
	}
	writeJSON(w, http.StatusOK, explorerBlocksResponse{Blocks: views, NextBefore: next})
}

func (s *Server) handleExplorerBlock(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	raw := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/explorer/blocks/"))
	if raw == "" || strings.Contains(raw, "/") {
		writeError(w, http.StatusBadRequest, "invalid block identifier")
		return
	}

	var block *blockchain.Block
	var found bool
	if height, err := strconv.ParseUint(raw, 10, 64); err == nil {
		block, found = s.chain.BlockByHeight(height)
	} else if len(raw) == 64 && isLowerHex(raw) {
		block, found = s.chain.BlockByHash(raw)
	} else {
		writeError(w, http.StatusBadRequest, "invalid block identifier")
		return
	}
	if !found || block == nil {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	writeJSON(w, http.StatusOK, explorerBlockView(block, true))
}

func (s *Server) handleExplorerTransactions(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	limit, ok := explorerLimit(w, r)
	if !ok {
		return
	}
	before, ok := explorerOptionalHeight(w, r, "before_height")
	if !ok {
		return
	}
	confirmed := s.chain.RecentTransactions(limit, before)
	views := make([]explorerTransactionResponse, 0, len(confirmed))
	chainHeight := s.chain.Height()
	for _, item := range confirmed {
		views = append(views, explorerConfirmedTransaction(item, chainHeight))
	}
	var next *uint64
	if len(confirmed) == limit && len(confirmed) > 0 {
		height := confirmed[len(confirmed)-1].BlockHeight
		next = &height
	}
	writeJSON(w, http.StatusOK, explorerTransactionsResponse{Transactions: views, NextBeforeHeight: next})
}

func (s *Server) handleExplorerTransaction(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	txID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/explorer/transactions/"))
	if len(txID) != 64 || strings.Contains(txID, "/") || !isLowerHex(txID) {
		writeError(w, http.StatusBadRequest, "invalid transaction ID")
		return
	}
	if tx, ok := s.node.MempoolTransaction(txID); ok && tx != nil {
		writeJSON(w, http.StatusOK, explorerTransactionResponse{
			Transaction: explorerTransaction(tx),
			Status:      "pending",
		})
		return
	}
	tx, block, ok := s.chain.TransactionByID(txID)
	if !ok || tx == nil || block == nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	height := block.Height
	confirmations := confirmationsAt(s.chain.Height(), height)
	writeJSON(w, http.StatusOK, explorerTransactionResponse{
		Transaction:    explorerTransaction(tx),
		Status:         "confirmed",
		BlockHeight:    &height,
		BlockHash:      block.Hash(),
		BlockTimestamp: block.Timestamp,
		Confirmations:  confirmations,
	})
}

func (s *Server) handleExplorerAddress(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	raw := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/explorer/addresses/"))
	if len(raw) != 40 || strings.Contains(raw, "/") || !isLowerHex(raw) {
		writeError(w, http.StatusBadRequest, "invalid account address")
		return
	}
	limit, ok := explorerLimit(w, r)
	if !ok {
		return
	}
	before, ok := explorerOptionalHeight(w, r, "before_height")
	if !ok {
		return
	}
	nextNonce, err := s.state.ExpectedNonce(raw)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	confirmed := s.chain.TransactionsForAddress(raw, limit, before)
	views := make([]explorerTransactionResponse, 0, len(confirmed))
	chainHeight := s.chain.Height()
	for _, item := range confirmed {
		views = append(views, explorerConfirmedTransaction(item, chainHeight))
	}
	var next *uint64
	if len(confirmed) == limit && len(confirmed) > 0 {
		height := confirmed[len(confirmed)-1].BlockHeight
		next = &height
	}
	writeJSON(w, http.StatusOK, explorerAddressResponse{
		Address:          raw,
		Balance:          s.state.Balance(raw),
		ConfirmedNonce:   s.state.AccountNonce(raw),
		NextNonce:        nextNonce,
		Transactions:     views,
		NextBeforeHeight: next,
	})
}

func (s *Server) handleExplorerSearch(w http.ResponseWriter, r *http.Request) {
	if !explorerGET(w, r) {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" || len(query) > 256 {
		writeError(w, http.StatusBadRequest, "invalid search query")
		return
	}

	if height, err := strconv.ParseUint(query, 10, 64); err == nil {
		if _, ok := s.chain.BlockByHeight(height); ok {
			writeJSON(w, http.StatusOK, explorerSearchResponse{Type: "block", Path: "/explorer/block?id=" + query})
			return
		}
		writeError(w, http.StatusNotFound, "search result not found")
		return
	}

	if len(query) == 64 && isLowerHex(query) {
		if _, ok := s.node.MempoolTransaction(query); ok {
			writeJSON(w, http.StatusOK, explorerSearchResponse{Type: "transaction", Path: "/explorer/tx?id=" + query})
			return
		}
		if _, _, ok := s.chain.TransactionByID(query); ok {
			writeJSON(w, http.StatusOK, explorerSearchResponse{Type: "transaction", Path: "/explorer/tx?id=" + query})
			return
		}
		if _, ok := s.chain.BlockByHash(query); ok {
			writeJSON(w, http.StatusOK, explorerSearchResponse{Type: "block", Path: "/explorer/block?id=" + query})
			return
		}
		writeError(w, http.StatusNotFound, "search result not found")
		return
	}

	if len(query) == 40 && isLowerHex(query) {
		writeJSON(w, http.StatusOK, explorerSearchResponse{Type: "address", Path: "/explorer/address?address=" + query})
		return
	}
	writeError(w, http.StatusNotFound, "search result not found")
}

func explorerGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return false
	}
	return true
}

func explorerLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := DefaultExplorerLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > MaxExplorerLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return 0, false
		}
		limit = parsed
	}
	return limit, true
}

func explorerOptionalHeight(w http.ResponseWriter, r *http.Request, key string) (*uint64, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, key+" must be an unsigned block height")
		return nil, false
	}
	return &value, true
}

func explorerTransaction(tx *transactions.Transaction) explorerTransactionView {
	if tx == nil {
		return explorerTransactionView{}
	}
	return explorerTransactionView{
		ID:     tx.ID,
		From:   tx.From,
		To:     tx.To,
		Amount: tx.Amount,
		Fee:    tx.Fee,
		Nonce:  tx.Nonce,
	}
}

func explorerBlockView(block *blockchain.Block, includeTransactions bool) explorerBlockResponse {
	if block == nil {
		return explorerBlockResponse{}
	}
	view := explorerBlockResponse{
		Height:           block.Height,
		Hash:             block.Hash(),
		Timestamp:        block.Timestamp,
		PreviousHash:     block.PreviousHash,
		MerkleRoot:       block.MerkleRoot,
		Difficulty:       block.Difficulty,
		Nonce:            block.Nonce,
		MinerAddress:     block.MinerAddress,
		TransactionCount: len(block.Transactions),
	}
	if includeTransactions && len(block.Transactions) > 0 {
		view.Transactions = make([]explorerTransactionView, 0, len(block.Transactions))
		for _, tx := range block.Transactions {
			if tx != nil {
				view.Transactions = append(view.Transactions, explorerTransaction(tx))
			}
		}
	}
	return view
}

func explorerConfirmedTransaction(item blockchain.ConfirmedTransaction, chainHeight uint64) explorerTransactionResponse {
	height := item.BlockHeight
	return explorerTransactionResponse{
		Transaction:    explorerTransaction(item.Transaction),
		Status:         "confirmed",
		BlockHeight:    &height,
		BlockHash:      item.BlockHash,
		BlockTimestamp: item.BlockTimestamp,
		Confirmations:  confirmationsAt(chainHeight, height),
	}
}

func confirmationsAt(chainHeight, blockHeight uint64) uint64 {
	if chainHeight < blockHeight {
		return 0
	}
	return chainHeight - blockHeight + 1
}
