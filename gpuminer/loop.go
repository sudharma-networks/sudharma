package gpuminer

import (
	"context"
	"fmt"
	"time"

	"github.com/sudharma-networks/sudharma/params"
)

type Backend interface {
	Name() string
	Search(ctx context.Context, work Work) (uint64, error)
}

type Loop struct {
	Client  *Client
	Address string
	Backend Backend
	Log     func(string, ...any)
	Sleep   func(time.Duration)
}

func (l *Loop) Run(ctx context.Context) (accepted int, err error) {
	if l == nil || l.Client == nil {
		return 0, fmt.Errorf("GPU miner client is unavailable")
	}
	if l.Backend == nil {
		return 0, fmt.Errorf("%s", params.GPUOnlyMiningMessage)
	}
	if err := params.ValidateMiningBackend(l.Backend.Name()); err != nil {
		return 0, err
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
			logf("GPU work fetch failed: %v", err)
			sleep(2 * time.Second)
			continue
		}
		logf("GPU mining %s height %d on %s …", work.Algorithm, work.Height, l.Backend.Name())
		nonce, err := l.Backend.Search(ctx, work)
		if err != nil {
			logf("GPU search failed: %v", err)
			sleep(time.Second)
			continue
		}
		result, err := l.Client.Submit(ctx, work, nonce)
		if err != nil {
			logf("GPU submit failed: %v", err)
			sleep(2 * time.Second)
			continue
		}
		status := result.Status
		if status == "" {
			status = "unknown"
		}
		if status != "accepted" {
			logf("GPU share not accepted: %s %s", status, result.Error)
			sleep(time.Second)
			continue
		}
		accepted++
		logf("accepted GPU share at height %d (total %d)", work.Height, accepted)
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
