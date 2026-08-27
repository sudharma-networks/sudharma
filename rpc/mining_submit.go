package rpc

import (
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/transactions"
)

type MiningSolution struct {
	WorkID          string `json:"work_id"`
	Nonce           uint64 `json:"nonce"`
	Algorithm       string `json:"algorithm"`
	Version         uint32 `json:"version"`
	Height          uint64 `json:"height"`
	Difficulty      uint32 `json:"difficulty"`
	TargetHex       string `json:"target"`
	HeaderPrefixHex string `json:"header_prefix"`
	RewardAddress   string `json:"reward_address"`
}

type MiningSubmitStatus string

const (
	MiningSubmitAccepted MiningSubmitStatus = "accepted"
	MiningSubmitInvalid  MiningSubmitStatus = "invalid"
	MiningSubmitStale    MiningSubmitStatus = "stale"
	MiningSubmitMutated  MiningSubmitStatus = "mutated"
)

type MiningSubmitResult struct {
	Status MiningSubmitStatus `json:"status"`
}

type MiningSolutionVerifier func(block *blockchain.Block, nonce uint64) bool

type activeMiningWork struct {
	template MiningWorkTemplate
	block    blockchain.Block
}

type MiningWorkService struct {
	mu       sync.RWMutex
	verifier MiningSolutionVerifier
	active   *activeMiningWork
}

func NewMiningWorkService(verifier MiningSolutionVerifier) *MiningWorkService {
	return &MiningWorkService{verifier: verifier}
}

func (s *MiningWorkService) Issue(block *blockchain.Block) (MiningWorkTemplate, error) {
	template, err := NewMiningWorkTemplate(block)
	if err != nil {
		return MiningWorkTemplate{}, err
	}

	snapshot := *block
	if block.Transactions != nil {
		snapshot.Transactions = append([]*transactions.Transaction(nil), block.Transactions...)
	}

	s.mu.Lock()
	s.active = &activeMiningWork{template: template, block: snapshot}
	s.mu.Unlock()
	return template, nil
}

func (s *MiningWorkService) Submit(solution MiningSolution) MiningSubmitResult {
	s.mu.RLock()
	active := s.active
	if active == nil || solution.WorkID != active.template.WorkID {
		s.mu.RUnlock()
		return MiningSubmitResult{Status: MiningSubmitStale}
	}
	if !solutionMatchesTemplate(solution, active.template) {
		s.mu.RUnlock()
		return MiningSubmitResult{Status: MiningSubmitMutated}
	}
	block := active.block
	verifier := s.verifier
	s.mu.RUnlock()

	if verifier == nil || !verifier(&block, solution.Nonce) {
		return MiningSubmitResult{Status: MiningSubmitInvalid}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil || s.active.template.WorkID != solution.WorkID {
		return MiningSubmitResult{Status: MiningSubmitStale}
	}
	s.active = nil
	return MiningSubmitResult{Status: MiningSubmitAccepted}
}

func solutionMatchesTemplate(solution MiningSolution, work MiningWorkTemplate) bool {
	return solution.Algorithm == work.Algorithm &&
		solution.Version == work.Version &&
		solution.Height == work.Height &&
		solution.Difficulty == work.Difficulty &&
		solution.TargetHex == work.TargetHex &&
		solution.HeaderPrefixHex == work.HeaderPrefixHex &&
		solution.RewardAddress == work.RewardAddress
}
