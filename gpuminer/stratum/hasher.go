package stratum

import (
	"context"
	"fmt"

	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/pool"
)

// CommandBackendShareMiner uses a Khushi GPU hasher subprocess for pool shares.
type CommandBackendShareMiner struct {
	Backend gpuminer.CommandBackend
}

func (m CommandBackendShareMiner) MineShare(ctx context.Context, job Job) (uint64, pool.ShareResult, error) {
	work := gpuminer.WorkFromCandidateBlock(&job.Block, job.PoolDifficulty, job.PoolTarget)
	nonce, err := m.Backend.Search(ctx, work)
	if err != nil {
		return 0, pool.ShareResult{}, err
	}
	result, err := pool.ValidateShare(&job.Block, nonce, job.PoolDifficulty, job.BlockDifficulty)
	if err != nil {
		return 0, pool.ShareResult{}, err
	}
	if result.Kind == pool.ShareInvalid {
		return 0, result, fmt.Errorf("GPU hasher nonce did not meet pool difficulty")
	}
	return nonce, result, nil
}

// NewShareMiner returns a GPU hasher miner when available, otherwise reference CPU search.
func NewShareMiner(backend gpuminer.CommandBackend) ShareMiner {
	if backend.Path != "" {
		return CommandBackendShareMiner{Backend: backend}
	}
	return ReferenceShareMiner{}
}
