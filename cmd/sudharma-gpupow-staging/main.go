package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/pow"
)

const (
	defaultListenAddress = "127.0.0.1:28646"
	stagingHeight        = uint64(0)
	stagingCacheNodes    = uint32(8)
)

// Mainnet rehearsal mode also exposes GET /v1/mining/staging/status via rehearsalAPI.
var stagingTarget = [32]byte{
	0x0f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

type stagingChallenge struct {
	ChallengeID  string `json:"challenge_id"`
	Algorithm    string `json:"algorithm"`
	Staging      bool   `json:"staging"`
	HeaderPrefix string `json:"header_prefix"`
	Target       string `json:"target"`
	Height       uint64 `json:"height"`
	CacheNodes   uint32 `json:"cache_nodes"`
	ProgramSeed  string `json:"program_seed,omitempty"`
	EpochSeed    string `json:"epoch_seed,omitempty"`
}

type stagingSubmission struct {
	Challenge stagingChallenge `json:"challenge"`
	Nonce     uint64           `json:"nonce"`
}

type stagingResult struct {
	Status string `json:"status"`
}

type stagingAPI struct {
	mu         sync.Mutex
	challenges map[string]stagingChallenge
	cache      []pow.GPUV1CacheNode
}

func newStagingAPI() *stagingAPI {
	epoch := pow.GPUV1EpochForHeight(stagingHeight)
	return &stagingAPI{
		challenges: make(map[string]stagingChallenge),
		cache:      pow.GPUV1BuildCache(pow.GPUV1EpochSeed(epoch), stagingCacheNodes),
	}
}

func (a *stagingAPI) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mining/staging/challenge", a.handleChallenge)
	mux.HandleFunc("/v1/mining/staging/submit", a.handleSubmit)
	return mux
}

func (a *stagingAPI) handleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	challenge, err := a.issueChallenge()
	if err != nil {
		http.Error(w, "unable to issue staging challenge", http.StatusInternalServerError)
		return
	}
	writeJSON(w, challenge)
}

func (a *stagingAPI) issueChallenge() (stagingChallenge, error) {
	var idBytes [16]byte
	var header [32]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return stagingChallenge{}, err
	}
	if _, err := rand.Read(header[:]); err != nil {
		return stagingChallenge{}, err
	}

	challenge := stagingChallenge{
		ChallengeID:  hex.EncodeToString(idBytes[:]),
		Algorithm:    pow.GPUV1AlgorithmID,
		Staging:      true,
		HeaderPrefix: hex.EncodeToString(header[:]),
		Target:       hex.EncodeToString(stagingTarget[:]),
		Height:       stagingHeight,
		CacheNodes:   stagingCacheNodes,
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.challenges[challenge.ChallengeID]; exists {
		return stagingChallenge{}, errors.New("challenge id collision")
	}
	a.challenges[challenge.ChallengeID] = challenge
	return challenge, nil
}

func (a *stagingAPI) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var submission stagingSubmission
	if err := decoder.Decode(&submission); err != nil {
		writeJSON(w, stagingResult{Status: "rejected"})
		return
	}

	if a.acceptSubmission(submission) {
		writeJSON(w, stagingResult{Status: "accepted"})
		return
	}
	writeJSON(w, stagingResult{Status: "rejected"})
}

func (a *stagingAPI) acceptSubmission(submission stagingSubmission) bool {
	a.mu.Lock()
	issued, ok := a.challenges[submission.Challenge.ChallengeID]
	if ok {
		delete(a.challenges, submission.Challenge.ChallengeID)
	}
	a.mu.Unlock()
	if !ok || issued != submission.Challenge {
		return false
	}

	header, err := hex.DecodeString(issued.HeaderPrefix)
	if err != nil {
		return false
	}
	target, err := hex.DecodeString(issued.Target)
	if err != nil || len(target) != 32 {
		return false
	}

	digest := pow.GPUV1ReferenceDigest(header, submission.Nonce, issued.Height, a.cache)
	return bytes.Compare(digest[:], target) <= 0
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func validateListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}
	if strings.TrimSpace(port) == "" {
		return errors.New("listen port is required")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("staging verifier must bind to loopback only")
	}
	return nil
}

func main() {
	listenAddress := flag.String("listen", defaultListenAddress, "localhost-only staging verifier listen address")
	mainnetRehearsal := flag.Bool("mainnet-rehearsal", false, "run isolated production-consensus mainnet mining rehearsal; public mainnet remains off")
	rehearsalBlocks := flag.Uint64("rehearsal-blocks", 50, "number of isolated mainnet rehearsal blocks (minimum 25)")
	flag.Parse()

	if err := validateListenAddress(*listenAddress); err != nil {
		log.Fatal(err)
	}

	var handler http.Handler
	if *mainnetRehearsal {
		api, err := newMainnetRehearsalAPI(*rehearsalBlocks)
		if err != nil {
			log.Fatal(err)
		}
		handler = api.handler()
		log.Printf(
			"Khushi mainnet rehearsal armed locally only: network=%s blocks=%d cache_nodes=%d; public mainnet launch/mining remain disabled",
			params.NetworkMainnet,
			*rehearsalBlocks,
			pow.GPUV1ProductionCacheNodes,
		)
	} else {
		handler = newStagingAPI().handler()
	}

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("Khushi localhost staging verifier listening on %s; consensus activation disabled on public networks", listener.Addr().String())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}