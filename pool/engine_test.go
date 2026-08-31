package pool

import (
	"context"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/gpuminer"
)

type fakeWorkSource struct {
	work   gpuminer.Work
	submit func(*blockchain.Block) error
}

func (f *fakeWorkSource) GetWork(context.Context, string) (gpuminer.Work, error) {
	return f.work, nil
}

func (f *fakeWorkSource) SubmitBlock(_ context.Context, block *blockchain.Block) (gpuminer.SubmitResult, error) {
	if f.submit != nil {
		if err := f.submit(block); err != nil {
			return gpuminer.SubmitResult{}, err
		}
	}
	return gpuminer.SubmitResult{Status: "accepted", Accepted: true}, nil
}

func TestEngineRefreshAndSubmitBlockShare(t *testing.T) {
	block := testBlock(t)
	block.Difficulty = 100
	source := &fakeWorkSource{
		work: gpuminer.Work{
			WorkID:      "work-1",
			Height:      block.Height,
			BlockReward: 10_000,
			Block:       block,
		},
		submit: func(got *blockchain.Block) error {
			if got.MinerAddress != "cccccccccccccccccccccccccccccccccccccccc" {
				t.Fatalf("payout address = %q", got.MinerAddress)
			}
			return nil
		},
	}
	cfg := DefaultConfig()
	cfg.PayoutAddress = "cccccccccccccccccccccccccccccccccccccccc"
	cfg.PayoutScheme = SchemePPLNS
	cfg.PoolDifficulty = 1
	cfg.RPCURL = "http://127.0.0.1:1"

	engine, err := NewEngine(cfg, source)
	if err != nil {
		t.Fatal(err)
	}
	job, err := engine.RefreshWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" {
		t.Fatal("expected job id")
	}

	nonce := findNonce(t, block, 100)
	worker, _ := ParseWorkerIdentity("dddddddddddddddddddddddddddddddddddddddd.rig")
	result, _, err := engine.SubmitShare(context.Background(), job, worker, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != ShareBlock {
		t.Fatalf("kind = %s", result.Kind)
	}
}
