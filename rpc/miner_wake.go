package rpc

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const DefaultMinerWakeURL = "http://127.0.0.1:28546/v1/wake"

func (s *Server) handleMinerWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	s.notifyDemandMinerWake()
	writeJSON(w, http.StatusAccepted, map[string]any{"awoken": true})
}

func (s *Server) notifyDemandMinerWake() {
	wakeURL := strings.TrimSpace(s.config.MinerWakeURL)
	if wakeURL == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, wakeURL, http.NoBody)
		if err != nil {
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}()
}
