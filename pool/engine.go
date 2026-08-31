package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/gpuminer"
)

// WorkSource fetches Sudharma candidate blocks for pool jobs.
type WorkSource interface {
	GetWork(ctx context.Context, address string) (gpuminer.Work, error)
	SubmitBlock(ctx context.Context, block *blockchain.Block) (gpuminer.SubmitResult, error)
}

// Job is a pool-visible mining job derived from a Sudharma candidate block.
type Job struct {
	ID              string
	WorkID          string
	Height          uint64
	Parent          string
	Timestamp       int64
	Version         uint32
	BlockDifficulty uint32
	PoolDifficulty  uint32
	PoolTarget      string
	BlockTarget     string
	BlockReward     uint64
	Block           blockchain.Block
	CreatedAt       time.Time
}

// Engine coordinates work refresh, share validation, payouts, and block submission.
type Engine struct {
	cfg    Config
	source WorkSource
	ledger *PayoutLedger

	mu       sync.RWMutex
	current  Job
	blockSeq uint64
}

func NewEngine(cfg Config, source WorkSource) (*Engine, error) {
	resolved, err := ResolveConfig(cfg)
	if err != nil {
		return nil, err
	}
	ledger, err := NewPayoutLedger(resolved.PayoutScheme, resolved.PPLNSWindow, resolved.PoolFeeBPS)
	if err != nil {
		return nil, err
	}
	return &Engine{
		cfg:    resolved,
		source: source,
		ledger: ledger,
	}, nil
}

func (e *Engine) Config() Config {
	if e == nil {
		return Config{}
	}
	return e.cfg
}

func (e *Engine) Ledger() *PayoutLedger {
	if e == nil {
		return nil
	}
	return e.ledger
}

func (e *Engine) CurrentJob() Job {
	if e == nil {
		return Job{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current
}

func (e *Engine) RefreshWork(ctx context.Context) (Job, error) {
	if e == nil || e.source == nil {
		return Job{}, fmt.Errorf("pool engine unavailable")
	}
	work, err := e.source.GetWork(ctx, e.cfg.PayoutAddress)
	if err != nil {
		return Job{}, err
	}
	if work.Block == nil {
		return Job{}, fmt.Errorf("mining work missing candidate block")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.blockSeq++
	job := Job{
		ID:              fmt.Sprintf("sudh-%d-%d", work.Height, e.blockSeq),
		WorkID:          work.WorkID,
		Height:          work.Height,
		Parent:          work.Parent,
		Timestamp:       work.Block.Timestamp,
		Version:         work.Block.Version,
		BlockDifficulty: work.Block.Difficulty,
		PoolDifficulty:  e.cfg.PoolDifficulty,
		PoolTarget:      TargetHex(e.cfg.PoolDifficulty),
		BlockTarget:     TargetHex(work.Block.Difficulty),
		BlockReward:     work.BlockReward,
		Block:           *work.Block,
		CreatedAt:       time.Now().UTC(),
	}
	e.current = job
	return job, nil
}

// SubmitShare validates a worker nonce against the given job and credits payouts.
func (e *Engine) SubmitShare(ctx context.Context, job Job, worker WorkerIdentity, nonce uint64) (ShareResult, ShareCredit, error) {
	if e == nil {
		return ShareResult{}, ShareCredit{}, fmt.Errorf("pool engine unavailable")
	}
	if err := ctx.Err(); err != nil {
		return ShareResult{}, ShareCredit{}, err
	}

	result, err := ValidateShare(&job.Block, nonce, job.PoolDifficulty, job.BlockDifficulty)
	if err != nil {
		return ShareResult{}, ShareCredit{}, err
	}
	if result.Kind == ShareInvalid {
		return result, ShareCredit{}, fmt.Errorf("share below pool difficulty")
	}

	credit := e.ledger.CreditShare(job.BlockReward, result, worker, job.ID, job.Height)
	if result.Kind == ShareBlock {
		block := job.Block
		block.Nonce = nonce
		block.MinerAddress = e.cfg.PayoutAddress
		if _, err := e.source.SubmitBlock(ctx, &block); err != nil {
			return result, credit, fmt.Errorf("block submit failed: %w", err)
		}
	}
	return result, credit, nil
}
