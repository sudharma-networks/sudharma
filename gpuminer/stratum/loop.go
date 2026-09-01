package stratum

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sudharma-networks/sudharma/pool"
)

// Loop mines Sudharma pool jobs received over Stratum v1.
type Loop struct {
	Client   *Client
	PoolURL  string
	Login    string
	Password string
	Miner    ShareMiner
	Once     bool
	Log      func(string, ...any)
	Sleep    func(time.Duration)
}

// ShareMiner searches for a valid pool share in a job template.
type ShareMiner interface {
	MineShare(ctx context.Context, job Job) (uint64, pool.ShareResult, error)
}

type ReferenceShareMiner struct{}

func (ReferenceShareMiner) MineShare(ctx context.Context, job Job) (uint64, pool.ShareResult, error) {
	block := job.Block
	const chunk = uint64(100_000)
	var start uint64
	for {
		if err := ctx.Err(); err != nil {
			return 0, pool.ShareResult{}, err
		}
		for i := uint64(0); i < chunk; i++ {
			nonce := start + i
			result, err := pool.ValidateShare(&block, nonce, job.PoolDifficulty, job.BlockDifficulty)
			if err != nil {
				return 0, pool.ShareResult{}, err
			}
			if result.Kind == pool.ShareValid || result.Kind == pool.ShareBlock {
				return nonce, result, nil
			}
		}
		start += chunk
	}
}

func (l *Loop) Run(ctx context.Context) (shares int, blocks int, err error) {
	if l == nil {
		return 0, 0, fmt.Errorf("stratum loop unavailable")
	}
	logf := l.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	sleep := l.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	miner := l.Miner
	if miner == nil {
		miner = ReferenceShareMiner{}
	}

	client := l.Client
	if client == nil {
		if l.PoolURL == "" || l.Login == "" {
			return 0, 0, fmt.Errorf("stratum pool URL and worker login are required")
		}
		password := l.Password
		if password == "" {
			password = "x"
		}
		client, err = Dial(ctx, l.PoolURL, l.Login, password)
		if err != nil {
			return 0, 0, err
		}
		defer client.Close()
	}

	for {
		if err := ctx.Err(); err != nil {
			return shares, blocks, nil
		}
		job, err := client.NextJob(ctx)
		if err != nil {
			if l.Once {
				return shares, blocks, err
			}
			logf("stratum job fetch failed: %v", err)
			sleep(2 * time.Second)
			continue
		}
		logf("pool job %s height %d pool_diff=%d", job.ID, job.Height, job.PoolDifficulty)

		nonce, result, err := miner.MineShare(ctx, job)
		if err != nil {
			if l.Once {
				return shares, blocks, err
			}
			logf("pool mine failed: %v", err)
			sleep(time.Second)
			continue
		}
		if err := client.SubmitShare(ctx, job.ID, nonce); err != nil {
			if l.Once {
				return shares, blocks, err
			}
			logf("pool share submit failed: %v", err)
			sleep(time.Second)
			continue
		}
		switch result.Kind {
		case pool.ShareBlock:
			blocks++
			logf("pool block share accepted at height %d", job.Height)
		case pool.ShareValid:
			shares++
			logf("pool share accepted at height %d", job.Height)
		}
		if l.Once {
			return shares, blocks, nil
		}
	}
}

// WorkerLogin builds wallet.worker from address and optional worker name.
func WorkerLogin(address, worker string) (string, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	login := address
	if worker != "" && worker != "default" {
		login = fmt.Sprintf("%s.%s", address, worker)
	}
	identity, err := pool.ParseWorkerIdentity(login)
	if err != nil {
		return "", err
	}
	return identity.Login, nil
}
