package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/pow"
)

const rehearsalMinerAddress = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type rehearsalStatus struct {
	Mode           string `json:"mode"`
	Network        string `json:"network"`
	ChainHeight    uint64 `json:"chain_height"`
	AcceptedBlocks uint64 `json:"accepted_blocks"`
	TargetBlocks   uint64 `json:"target_blocks"`
	IssuedSupply   uint64 `json:"issued_supply"`
	Completed      bool   `json:"completed"`
}

type rehearsalAPI struct {
	mu             sync.Mutex
	chain          *blockchain.Chain
	state          *blockchain.State
	targetBlocks   uint64
	acceptedBlocks uint64
	challenge      *stagingChallenge
	candidate      *blockchain.Block
}

func newMainnetRehearsalAPI(targetBlocks uint64) (*rehearsalAPI, error) {
	if targetBlocks < 25 {
		return nil, fmt.Errorf("mainnet rehearsal requires at least 25 blocks")
	}
	if targetBlocks > 1000 {
		return nil, fmt.Errorf("mainnet rehearsal block count is unreasonably large")
	}
	if params.MainnetLaunchAuthorized || params.MainnetMiningAuthorized {
		return nil, fmt.Errorf("isolated mainnet rehearsal requires public mainnet launch and mining gates to remain closed")
	}
	if params.GPUV1MainnetActivationHeight != params.GPUV1ActivationDisabled {
		return nil, fmt.Errorf("isolated mainnet rehearsal requires public mainnet GPU activation to remain disabled")
	}

	policy := blockchain.PoWPolicy{GPUV1ActivationHeight: 1}
	verifier, err := pow.NewChainProofVerifier(policy)
	if err != nil {
		return nil, fmt.Errorf("construct mainnet rehearsal proof verifier: %w", err)
	}
	chain, err := newValidationOnlyMainnetChain(policy, verifier)
	if err != nil {
		return nil, err
	}

	return &rehearsalAPI{
		chain:        chain,
		state:        blockchain.NewStateFor(params.MonetaryPolicyMainnet),
		targetBlocks: targetBlocks,
	}, nil
}

func newValidationOnlyMainnetChain(policy blockchain.PoWPolicy, verifier blockchain.ProofVerifier) (*blockchain.Chain, error) {
	encoded, err := json.Marshal([]*blockchain.Block{blockchain.NewMainnetGenesisBlock()})
	if err != nil {
		return nil, fmt.Errorf("encode mainnet rehearsal genesis: %w", err)
	}

	file, err := os.CreateTemp("", "sudharma-mainnet-rehearsal-*.json")
	if err != nil {
		return nil, fmt.Errorf("create mainnet rehearsal chain file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close mainnet rehearsal chain file: %w", err)
	}
	defer os.Remove(path)
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		return nil, fmt.Errorf("write mainnet rehearsal chain file: %w", err)
	}

	chain, err := blockchain.LoadChainFromFileForWithConsensus(
		path,
		params.NetworkMainnet,
		policy,
		verifier,
	)
	if err != nil {
		return nil, fmt.Errorf("construct validation-only mainnet rehearsal chain: %w", err)
	}
	return chain, nil
}

func (a *rehearsalAPI) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mining/staging/challenge", a.handleChallenge)
	mux.HandleFunc("/v1/mining/staging/submit", a.handleSubmit)
	mux.HandleFunc("/v1/mining/staging/status", a.handleStatus)
	return mux
}

func (a *rehearsalAPI) handleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.acceptedBlocks >= a.targetBlocks {
		writeJSON(w, a.statusLocked())
		return
	}
	if a.challenge != nil {
		writeJSON(w, *a.challenge)
		return
	}

	challenge, candidate, err := a.issueChallengeLocked()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.challenge = &challenge
	a.candidate = candidate
	writeJSON(w, challenge)
}

func (a *rehearsalAPI) issueChallengeLocked() (stagingChallenge, *blockchain.Block, error) {
	tip := a.chain.Tip()
	if tip == nil {
		return stagingChallenge{}, nil, fmt.Errorf("mainnet rehearsal chain has no tip")
	}
	height := tip.Height + 1
	difficulty, err := blockchain.ExpectedNextDifficulty(a.chain)
	if err != nil {
		return stagingChallenge{}, nil, err
	}

	candidate := &blockchain.Block{
		Version:      2,
		Height:       height,
		Timestamp:    tip.Timestamp + int64(params.TargetBlockTimeSeconds),
		PreviousHash: tip.Hash(),
		Difficulty:   difficulty,
		MinerAddress: rehearsalMinerAddress,
	}
	candidate.UpdateMerkleRoot()
	header := candidate.HeaderBytes(0)
	if len(header) < 8 {
		return stagingChallenge{}, nil, fmt.Errorf("candidate header is too short")
	}

	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return stagingChallenge{}, nil, err
	}

	target := pow.TargetFromDifficulty(difficulty)
	var targetBytes [32]byte
	target.FillBytes(targetBytes[:])
	programSeed := pow.GPUV1ProgramSeed(pow.GPUV1ProgramForHeight(height))
	epochSeed := pow.GPUV1EpochSeed(pow.GPUV1EpochForHeight(height))

	challenge := stagingChallenge{
		ChallengeID:  hex.EncodeToString(idBytes[:]),
		Algorithm:    pow.GPUV1AlgorithmID,
		Staging:      true,
		HeaderPrefix: hex.EncodeToString(header[:len(header)-8]),
		Target:       hex.EncodeToString(targetBytes[:]),
		Height:       height,
		CacheNodes:   pow.GPUV1ProductionCacheNodes,
		ProgramSeed:  hex.EncodeToString(programSeed[:]),
		EpochSeed:    hex.EncodeToString(epochSeed[:]),
	}
	return challenge, candidate, nil
}

func (a *rehearsalAPI) handleSubmit(w http.ResponseWriter, r *http.Request) {
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

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.challenge == nil || a.candidate == nil || *a.challenge != submission.Challenge {
		writeJSON(w, stagingResult{Status: "rejected"})
		return
	}

	candidate := *a.candidate
	candidate.Nonce = submission.Nonce
	workingState := a.state.Clone()
	if _, err := blockchain.ProcessBlockFor(
		workingState,
		params.MonetaryPolicyMainnet,
		&candidate,
		rehearsalMinerAddress,
	); err != nil {
		writeJSON(w, stagingResult{Status: "rejected"})
		return
	}
	if err := a.chain.AddBlock(&candidate); err != nil {
		writeJSON(w, stagingResult{Status: "rejected"})
		return
	}
	if err := a.state.ReplaceWith(workingState); err != nil {
		panic(fmt.Sprintf("mainnet rehearsal state commit failed after accepted block: %v", err))
	}

	a.acceptedBlocks++
	a.challenge = nil
	a.candidate = nil
	writeJSON(w, stagingResult{Status: "accepted"})
}

func (a *rehearsalAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	writeJSON(w, a.statusLocked())
}

func (a *rehearsalAPI) statusLocked() rehearsalStatus {
	return rehearsalStatus{
		Mode:           "mainnet-rehearsal",
		Network:        string(params.NetworkMainnet),
		ChainHeight:    a.chain.Height(),
		AcceptedBlocks: a.acceptedBlocks,
		TargetBlocks:   a.targetBlocks,
		IssuedSupply:   a.state.IssuedSupply(),
		Completed:      a.acceptedBlocks == a.targetBlocks,
	}
}
