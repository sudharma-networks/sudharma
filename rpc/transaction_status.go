package rpc

import (
	"net/http"
	"strings"

	"github.com/sudharma-networks/sudharma/transactions"
)

type transactionStatusResponse struct {
	Transaction   *transactions.Transaction `json:"transaction"`
	Status        string                    `json:"status"`
	BlockHeight   *uint64                   `json:"block_height,omitempty"`
	BlockHash     string                    `json:"block_hash,omitempty"`
	Confirmations uint64                    `json:"confirmations"`
}

func (s *Server) handleTransactionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	txID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/transactions/"))
	if len(txID) != 64 || strings.Contains(txID, "/") || !isLowerHex(txID) {
		writeError(w, http.StatusBadRequest, "invalid transaction ID")
		return
	}

	if tx, ok := s.node.MempoolTransaction(txID); ok {
		writeJSON(w, http.StatusOK, transactionStatusResponse{
			Transaction: tx,
			Status:      "pending",
		})
		return
	}

	tx, block, ok := s.chain.TransactionByID(txID)
	if !ok || block == nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}

	height := block.Height
	chainHeight := s.chain.Height()
	confirmations := uint64(1)
	if chainHeight >= height {
		confirmations = chainHeight - height + 1
	}
	writeJSON(w, http.StatusOK, transactionStatusResponse{
		Transaction:   tx,
		Status:        "confirmed",
		BlockHeight:   &height,
		BlockHash:     block.Hash(),
		Confirmations: confirmations,
	})
}

func isLowerHex(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
