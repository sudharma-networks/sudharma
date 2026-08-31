package pool

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// PayoutScheme selects how a pool distributes rewards to workers.
type PayoutScheme string

const (
	SchemeSolo  PayoutScheme = "solo"
	SchemePPS   PayoutScheme = "pps"
	SchemePPLNS PayoutScheme = "pplns"
	SchemeFPPS  PayoutScheme = "fpps"
)

func (s PayoutScheme) String() string {
	return string(s)
}

// NormalizePayoutScheme validates and normalizes operator input.
func NormalizePayoutScheme(raw string) (PayoutScheme, error) {
	switch PayoutScheme(strings.ToLower(strings.TrimSpace(raw))) {
	case "", SchemePPLNS:
		return SchemePPLNS, nil
	case SchemeSolo:
		return SchemeSolo, nil
	case SchemePPS:
		return SchemePPS, nil
	case SchemeFPPS:
		return SchemeFPPS, nil
	default:
		return "", fmt.Errorf("unsupported payout scheme %q (use solo, pps, pplns, or fpps)", raw)
	}
}

// ShareCredit records one accepted share for payout accounting.
type ShareCredit struct {
	Worker  WorkerIdentity
	Work    uint64
	Value   uint64
	JobID   string
	Height  uint64
	IsBlock bool
}

// PayoutLedger tracks share credits and block rewards for pool operators.
type PayoutLedger struct {
	mu sync.Mutex

	scheme      PayoutScheme
	window      int
	poolFeeBPS  uint64
	balances    map[string]uint64
	pendingPPLN []ShareCredit
	blocksFound uint64
	sharesSeen  uint64
}

func NewPayoutLedger(scheme PayoutScheme, pplnsWindow int, poolFeeBPS uint64) (*PayoutLedger, error) {
	if _, err := NormalizePayoutScheme(string(scheme)); err != nil {
		return nil, err
	}
	if pplnsWindow <= 0 {
		pplnsWindow = 10_000
	}
	if poolFeeBPS > 10_000 {
		return nil, fmt.Errorf("pool fee must be <= 10000 bps")
	}
	return &PayoutLedger{
		scheme:     scheme,
		window:     pplnsWindow,
		poolFeeBPS: poolFeeBPS,
		balances:   make(map[string]uint64),
	}, nil
}

func (l *PayoutLedger) Scheme() PayoutScheme {
	if l == nil {
		return SchemePPLNS
	}
	return l.scheme
}

func (l *PayoutLedger) Balance(address string) uint64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.balances[stringsLower(address)]
}

func (l *PayoutLedger) CreditShare(blockReward uint64, share ShareResult, worker WorkerIdentity, jobID string, height uint64) ShareCredit {
	if l == nil {
		return ShareCredit{}
	}

	credit := ShareCredit{
		Worker: worker,
		Work:   share.ShareWork,
		Value:  applyFee(ShareValue(blockReward, share.PoolDifficulty, share.BlockDifficulty), l.poolFeeBPS),
		JobID:  jobID,
		Height: height,
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.sharesSeen++

	switch l.scheme {
	case SchemeSolo:
		if share.Kind == ShareBlock {
			l.balances[worker.Address] += applyFee(blockReward, l.poolFeeBPS)
		}
	case SchemePPS, SchemeFPPS:
		if share.Kind == ShareValid || share.Kind == ShareBlock {
			l.balances[worker.Address] += credit.Value
		}
		if l.scheme == SchemeFPPS && share.Kind == ShareBlock {
			l.balances[worker.Address] += applyFee(blockReward, l.poolFeeBPS)
		}
	case SchemePPLNS:
		if share.Kind == ShareValid || share.Kind == ShareBlock {
			l.pendingPPLN = append(l.pendingPPLN, credit)
			if len(l.pendingPPLN) > l.window {
				l.pendingPPLN = l.pendingPPLN[len(l.pendingPPLN)-l.window:]
			}
		}
		if share.Kind == ShareBlock {
			l.distributePPLNSLocked(blockReward)
		}
	}

	if share.Kind == ShareBlock {
		credit.IsBlock = true
		l.blocksFound++
	}
	return credit
}

func (l *PayoutLedger) distributePPLNSLocked(blockReward uint64) {
	if len(l.pendingPPLN) == 0 {
		return
	}
	var totalWork uint64
	for _, item := range l.pendingPPLN {
		totalWork += item.Work
	}
	if totalWork == 0 {
		return
	}
	reward := applyFee(blockReward, l.poolFeeBPS)
	for _, item := range l.pendingPPLN {
		shareReward := uint64(math.Floor(float64(reward) * float64(item.Work) / float64(totalWork)))
		l.balances[item.Worker.Address] += shareReward
	}
}

func applyFee(amount uint64, feeBPS uint64) uint64 {
	if amount == 0 || feeBPS == 0 {
		return amount
	}
	fee := (amount * feeBPS) / 10_000
	if fee >= amount {
		return 0
	}
	return amount - fee
}

func stringsLower(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
