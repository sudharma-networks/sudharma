package gpuminer

import (
	"context"
	"fmt"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
)

type Backend interface {
	Name() string
	Search(ctx context.Context, work Work) (uint64, error)
}

type Loop struct {
	Client  RPCClient
	Address string
	Backend Backend
	Once    bool
	Log     func(string, ...any)
	Sleep   func(time.Duration)
}

func (l *Loop) Run(ctx context.Context) (accepted int, err error) {
	if l == nil || l.Client == nil {
		return 0, fmt.Errorf("GPU miner client is unavailable")
	}
	if l.Backend != nil {
		if err := params.ValidateMiningBackend(l.Backend.Name()); err != nil {
			return 0, err
		}
	}
	logf := l.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	sleep := l.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}

	for {
		if err := ctx.Err(); err != nil {
			return accepted, nil
		}
		work, err := l.Client.GetWork(ctx, l.Address)
		if err != nil {
			if l.Once {
				return accepted, err
			}
			logf("GPU work fetch failed: %v", err)
			sleep(2 * time.Second)
			continue
		}

		ok, err := l.handleWork(ctx, work, logf)
		if err != nil {
			if l.Once {
				return accepted, err
			}
			logf("GPU mine failed: %v", err)
			sleep(time.Second)
			continue
		}
		if ok {
			accepted++
		}
		if l.Once {
			return accepted, nil
		}
		if !ok {
			sleep(time.Second)
		}
	}
}

func (l *Loop) handleWork(ctx context.Context, work Work, logf func(string, ...any)) (bool, error) {
	if work.Block != nil {
		logf("GPU miner candidate height %d for %s (demand miner is a separate process)", work.Height, work.RewardAddress)
		mined, err := mineCandidateBlock(ctx, work.Block)
		if err != nil {
			return false, err
		}
		result, err := l.Client.SubmitBlock(ctx, mined)
		if err != nil {
			return false, err
		}
		return l.accepted(result, work.Height, logf)
	}

	if l.Backend == nil {
		return false, fmt.Errorf("%s", params.GPUOnlyMiningMessage)
	}
	logf("GPU mining %s height %d on %s …", work.Algorithm, work.Height, l.Backend.Name())
	nonce, err := l.Backend.Search(ctx, work)
	if err != nil {
		return false, err
	}
	result, err := l.Client.Submit(ctx, work, nonce)
	if err != nil {
		return false, err
	}
	return l.accepted(result, work.Height, logf)
}

func (l *Loop) accepted(result SubmitResult, height uint64, logf func(string, ...any)) (bool, error) {
	status := result.Status
	if status == "" && result.Accepted {
		status = "accepted"
	}
	if status == "" {
		status = "unknown"
	}
	if status != "accepted" {
		logf("GPU share not accepted: %s %s", status, result.Error)
		return false, nil
	}
	logf("accepted GPU share at height %d balance=%d", height, result.Balance)
	return true, nil
}

func mineCandidateBlock(ctx context.Context, candidate *blockchain.Block) (*blockchain.Block, error) {
	if candidate == nil {
		return nil, fmt.Errorf("missing candidate block")
	}
	copy := *candidate
	var start uint64
	const chunk = uint64(100_000)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result := miner.Mine(&copy, start, chunk)
		if result.Found {
			return result.Block, nil
		}
		start += chunk
		if start >= 50_000_000 {
			return nil, fmt.Errorf("no valid nonce found")
		}
	}
}

type StaticNonceBackend struct {
	BackendName string
	Nonce       uint64
}

func (s StaticNonceBackend) Name() string {
	if s.BackendName == "" {
		return params.ProductionMiningBackend
	}
	return s.BackendName
}

func (s StaticNonceBackend) Search(context.Context, Work) (uint64, error) {
	if err := params.ValidateMiningBackend(s.Name()); err != nil {
		return 0, err
	}
	return s.Nonce, nil
}

type RejectedBackend struct {
	BackendName string
}

func (r RejectedBackend) Name() string { return r.BackendName }

func (RejectedBackend) Search(context.Context, Work) (uint64, error) {
	return 0, fmt.Errorf("%s", params.GPUOnlyMiningMessage)
}
