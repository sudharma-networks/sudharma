package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

var gpuMinerAddressPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// GPU mining HTTP API — independent of demandminer.
//
//	GET/POST /v1/mining/work     candidate block for the caller's wallet
//	POST     /v1/mining/submit   AcceptBlock + BroadcastBlock
//
// Demand miner (sudharma-demand-miner / sudharmad -mineblocks) is unchanged
// and can run in parallel. Both compete on the same tip; the first accepted
// block wins. This API never starts or configures the demand miner.
func (s *Server) handleMiningWork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodGet+", "+http.MethodPost)
		return
	}
	if s.node == nil || s.chain == nil || s.state == nil {
		writeError(w, http.StatusServiceUnavailable, "chain not ready")
		return
	}

	address, err := gpuMinerAddressFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	block, err := candidateGPUMinerBlock(s.chain, s.node.Mempool(), address)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	tip := s.chain.Tip()
	parent := ""
	if tip != nil {
		parent = tip.Hash()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"work_id":        fmt.Sprintf("gpu-candidate-%d-%s", block.Height, address[:8]),
		"job":            "candidate-block",
		"algorithm":      params.ProductionMiningAlgorithm,
		"backend":        params.ProductionMiningBackend,
		"status":         "ready",
		"version":        0,
		"height":         block.Height,
		"parent":         parent,
		"difficulty":     block.Difficulty,
		"target":         consensus.TargetFromDifficulty(block.Difficulty).Text(16),
		"timestamp":      block.Timestamp,
		"reward_address": address,
		"block":          block,
		"block_reward":   consensus.BlockSubsidy(block.Height),
		"miner_balance":  s.state.Balance(address),
		"pow_compat":     buildPOWCompatWork(block, parent, consensus.BlockSubsidy(block.Height)),
		"note":           "GPU miner API. Demand miner is a separate process and is not modified. Submit the solved block to /v1/mining/submit to credit reward_address.",
	})
}

func (s *Server) handleMiningSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if s.node == nil || s.chain == nil || s.state == nil {
		writeError(w, http.StatusServiceUnavailable, "chain not ready")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mining submit body")
		return
	}

	var probe struct {
		Algorithm    string `json:"algorithm"`
		MinerAddress string `json:"MinerAddress"`
		PreviousHash string `json:"PreviousHash"`
		Height       uint64 `json:"Height"`
	}
	_ = json.Unmarshal(raw, &probe)

	if strings.TrimSpace(probe.Algorithm) != "" {
		if err := params.ValidateProductionMiningAlgorithm(probe.Algorithm); err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
	}

	if strings.TrimSpace(probe.MinerAddress) != "" || strings.TrimSpace(probe.PreviousHash) != "" || probe.Height > 0 {
		var block blockchain.Block
		if err := json.Unmarshal(raw, &block); err != nil {
			writeError(w, http.StatusBadRequest, "invalid block JSON: "+err.Error())
			return
		}
		s.acceptGPUMinerBlock(w, &block)
		return
	}

	writeError(w, http.StatusServiceUnavailable, params.GPUOnlyMiningMessage+" GPU-PoW work is not active on this node yet.")
}

func (s *Server) acceptGPUMinerBlock(w http.ResponseWriter, block *blockchain.Block) {
	if block == nil {
		writeError(w, http.StatusBadRequest, "block cannot be nil")
		return
	}
	block.MinerAddress = strings.ToLower(strings.TrimSpace(block.MinerAddress))
	if err := validateGPUMinerAddress(block.MinerAddress); err != nil {
		writeError(w, http.StatusBadRequest, "miner_address: "+err.Error())
		return
	}

	if err := s.node.AcceptBlock(block); err != nil {
		writeError(w, http.StatusBadRequest, "block rejected: "+err.Error())
		return
	}

	broadcastNote := ""
	if err := s.node.BroadcastBlock(block); err != nil {
		broadcastNote = err.Error()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "accepted",
		"accepted":       true,
		"height":         block.Height,
		"hash":           block.Hash(),
		"reward_address": block.MinerAddress,
		"balance":        s.state.Balance(block.MinerAddress),
		"broadcast":      broadcastNote,
	})
}

func gpuMinerAddressFromRequest(r *http.Request) (string, error) {
	address := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("address")))
	if address == "" && r.Method == http.MethodPost && r.Body != nil {
		var body struct {
			Address string `json:"address"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		address = strings.ToLower(strings.TrimSpace(body.Address))
	}
	if address == "" {
		return "", fmt.Errorf("address required")
	}
	if err := validateGPUMinerAddress(address); err != nil {
		return "", err
	}
	return address, nil
}

func validateGPUMinerAddress(address string) error {
	got := strings.ToLower(strings.TrimSpace(address))
	if !gpuMinerAddressPattern.MatchString(got) {
		return fmt.Errorf("wallet address must be 40 lowercase hex characters")
	}
	return nil
}

func candidateGPUMinerBlock(chain *blockchain.Chain, pool *mempool.Mempool, minerAddr string) (*blockchain.Block, error) {
	if chain == nil {
		return nil, fmt.Errorf("chain not ready")
	}
	if pool == nil {
		return nil, fmt.Errorf("mempool not ready")
	}
	tip := chain.Tip()
	if tip == nil {
		return nil, fmt.Errorf("chain tip not ready")
	}
	block, err := blockchain.NewBlockFromMempool(tip, pool)
	if err != nil {
		return nil, err
	}
	if block.Timestamp <= tip.Timestamp {
		block.Timestamp = tip.Timestamp + 1
	}
	difficulty, err := blockchain.ExpectedNextDifficulty(chain)
	if err != nil {
		return nil, err
	}
	block.Difficulty = difficulty
	block.MinerAddress = minerAddr
	block.Nonce = 0
	block.UpdateMerkleRoot()
	return block, nil
}
