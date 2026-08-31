package demandminer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const DefaultWakeListen = "127.0.0.1:28546"

type WakeSignaler interface {
	Wake()
}

type WakeSleeper struct {
	mu   sync.Mutex
	wake chan struct{}
}

func NewWakeSleeper() *WakeSleeper {
	return &WakeSleeper{wake: make(chan struct{}, 1)}
}

func (s *WakeSleeper) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *WakeSleeper) Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	case <-s.wake:
		return nil
	}
}

type WakeServer struct {
	sleeper WakeSignaler
	server  *http.Server
}

func NewWakeServer(listen string, sleeper WakeSignaler) (*WakeServer, error) {
	if strings.TrimSpace(listen) == "" {
		listen = DefaultWakeListen
	}
	if err := validateLoopbackHostPort(listen); err != nil {
		return nil, fmt.Errorf("wake_listen: %w", err)
	}
	if sleeper == nil {
		return nil, errors.New("wake signaler is required")
	}

	ws := &WakeServer{sleeper: sleeper}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", ws.handleHealth)
	mux.HandleFunc("/v1/wake", ws.handleWake)
	ws.server = &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       15 * time.Second,
	}
	return ws, nil
}

func (ws *WakeServer) ListenAndServe() error {
	return ws.server.ListenAndServe()
}

func (ws *WakeServer) Serve(listener net.Listener) error {
	return ws.server.Serve(listener)
}

func (ws *WakeServer) Shutdown(ctx context.Context) error {
	if ws == nil || ws.server == nil {
		return nil
	}
	return ws.server.Shutdown(ctx)
}

func (ws *WakeServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeWakeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	writeWakeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (ws *WakeServer) handleWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeWakeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	ws.sleeper.Wake()
	writeWakeJSON(w, http.StatusAccepted, map[string]any{"awoken": true})
}

func writeWakeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func validateLoopbackHostPort(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" || port == "" {
		return errors.New("must be host:port")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("host must be loopback")
	}
	parsed, err := url.Parse("http://" + value)
	if err != nil || parsed.Host == "" {
		return errors.New("must be a valid listen address")
	}
	return nil
}
