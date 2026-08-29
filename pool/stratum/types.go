package stratum

import "context"

const (
	maxMessageBytes  = 64 * 1024
	maxIdentityBytes = 128
	maxWorkerBytes   = 32
)

type WorkerIdentity struct {
	Wallet string
	Worker string
}

type Work struct {
	WorkID          string
	Algorithm       string
	TargetHex       string
	HeaderPrefixHex string
	RewardAddress   string
	Version         uint32
	Height          uint64
	Difficulty      uint32
}

type Candidate struct {
	Work     Work
	JobID    string
	Identity WorkerIdentity
	Lane     uint32
	Nonce    uint64
}

type SourceResult string

const (
	SourceAccepted SourceResult = "accepted"
	SourceInvalid  SourceResult = "invalid"
	SourceStale    SourceResult = "stale"
	SourceMutated  SourceResult = "mutated"
)

type WorkSource interface {
	CurrentWork(context.Context, string) (Work, error)
	Submit(context.Context, Candidate) (SourceResult, error)
}

type ShareVerifier interface {
	MeetsTarget(context.Context, Work, uint64, [32]byte) (bool, error)
}

type SubmitStatus string
