package rpc

import (
	"net/http"

	"github.com/sudharma-networks/sudharma/params"
)

func (s *Server) handleMiningWork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPost)
		return
	}
	writeError(w, http.StatusServiceUnavailable, params.GPUOnlyMiningMessage+" GPU-PoW work is not active on this node yet.")
}

func (s *Server) handleMiningSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	writeError(w, http.StatusServiceUnavailable, params.GPUOnlyMiningMessage+" GPU-PoW work is not active on this node yet.")
}
