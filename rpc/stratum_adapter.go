package rpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/pool/stratum"
)

type MiningBlockProvider interface {
	MiningBlock(context.Context) (*blockchain.Block, error)
}

type StratumWorkSource struct {
	mu       sync.RWMutex
	service  *MiningWorkService
	provider MiningBlockProvider
	current  map[string]MiningWorkTemplate
}

func NewStratumWorkSource(service *MiningWorkService, provider MiningBlockProvider) (*StratumWorkSource, error) {
	if service == nil {
		return nil, errors.New("mining work service is required")
	}
	if provider == nil {
		return nil, errors.New("mining block provider is required")
	}
	return &StratumWorkSource{service: service, provider: provider, current: make(map[string]MiningWorkTemplate, 1)}, nil
}

func (s *StratumWorkSource) CurrentWork(ctx context.Context, rewardAddress string) (stratum.Work, error) {
	if err := ctx.Err(); err != nil {
		return stratum.Work{}, err
	}
	if rewardAddress == "" {
		return stratum.Work{}, errors.New("mining reward address is required")
	}
	block, err := s.provider.MiningBlock(ctx)
	if err != nil {
		return stratum.Work{}, fmt.Errorf("get mining block: %w", err)
	}
	if block == nil {
		return stratum.Work{}, errors.New("mining block provider returned nil block")
	}
	copied := *block
	copied.MinerAddress = rewardAddress
	template, err := s.service.Issue(&copied)
	if err != nil {
		return stratum.Work{}, fmt.Errorf("issue mining work: %w", err)
	}
	s.mu.Lock()
	s.current = map[string]MiningWorkTemplate{template.WorkID: template}
	s.mu.Unlock()
	return stratumWork(template), nil
}

func (s *StratumWorkSource) Submit(ctx context.Context, candidate stratum.Candidate) (stratum.SourceResult, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.RLock()
	template, ok := s.current[candidate.Work.WorkID]
	s.mu.RUnlock()
	if !ok {
		return stratum.SourceStale, nil
	}
	result := s.service.Submit(MiningSolution{
		WorkID: template.WorkID, Nonce: candidate.Nonce, Algorithm: template.Algorithm,
		Version: template.Version, Height: template.Height, Difficulty: template.Difficulty,
		TargetHex: template.TargetHex, HeaderPrefixHex: template.HeaderPrefixHex, RewardAddress: template.RewardAddress,
	})
	switch result.Status {
	case MiningSubmitAccepted:
		return stratum.SourceAccepted, nil
	case MiningSubmitInvalid:
		return stratum.SourceInvalid, nil
	case MiningSubmitStale:
		return stratum.SourceStale, nil
	case MiningSubmitMutated:
		return stratum.SourceMutated, nil
	default:
		return "", fmt.Errorf("unknown mining submit status %q", result.Status)
	}
}

func stratumWork(t MiningWorkTemplate) stratum.Work {
	return stratum.Work{WorkID: t.WorkID, Algorithm: t.Algorithm, Version: t.Version, Height: t.Height, Difficulty: t.Difficulty, TargetHex: t.TargetHex, HeaderPrefixHex: t.HeaderPrefixHex, RewardAddress: t.RewardAddress}
}
