package pool_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/gpuminer"
	minerstratum "github.com/sudharma-networks/sudharma/gpuminer/stratum"
	"github.com/sudharma-networks/sudharma/pool"
	poolstratum "github.com/sudharma-networks/sudharma/pool/stratum"
)

type stubWorkSource struct {
	work gpuminer.Work
}

func (s stubWorkSource) GetWork(context.Context, string) (gpuminer.Work, error) {
	return s.work, nil
}

func (s stubWorkSource) SubmitBlock(context.Context, *blockchain.Block) (gpuminer.SubmitResult, error) {
	return gpuminer.SubmitResult{Status: "accepted", Accepted: true}, nil
}

func TestStratumPoolRoundTripCreditsWorkerShare(t *testing.T) {
	genesis := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(genesis, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	block.Difficulty = 1
	block.MinerAddress = "cccccccccccccccccccccccccccccccccccccccc"
	block.UpdateMerkleRoot()

	source := stubWorkSource{
		work: gpuminer.Work{
			WorkID:      "work-e2e",
			Height:      block.Height,
			BlockReward: 10_000,
			Block:       block,
		},
	}

	cfg := pool.DefaultConfig()
	cfg.PayoutAddress = block.MinerAddress
	cfg.PayoutScheme = pool.SchemePPS
	cfg.PoolDifficulty = 1
	cfg.RPCURL = "http://127.0.0.1:1"

	engine, err := pool.NewEngine(cfg, source)
	if err != nil {
		t.Fatal(err)
	}

	server := poolstratum.NewServer(engine, func(string, ...any) {})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = server.Serve(ctx, listener)
	}()
	time.Sleep(20 * time.Millisecond)

	worker, err := pool.ParseWorkerIdentity("dddddddddddddddddddddddddddddddddddddddd.rig")
	if err != nil {
		t.Fatal(err)
	}
	login, err := minerstratum.WorkerLogin(worker.Address, worker.WorkerName)
	if err != nil {
		t.Fatal(err)
	}

	loopCtx, loopCancel := context.WithTimeout(ctx, 10*time.Second)
	defer loopCancel()
	loop := &minerstratum.Loop{
		PoolURL:  listener.Addr().String(),
		Login:    login,
		Password: "x",
		Once:     true,
	}
	shares, blocks, err := loop.Run(loopCtx)
	if err != nil {
		t.Fatal(err)
	}
	if shares+blocks < 1 {
		t.Fatalf("expected accepted share, shares=%d blocks=%d", shares, blocks)
	}
	if engine.Ledger().Balance(worker.Address) == 0 {
		t.Fatal("expected PPS ledger credit for worker")
	}
}
