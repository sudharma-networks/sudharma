package stratum_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/pool"
	"github.com/sudharma-networks/sudharma/pool/stratum"
)

type stubEngine struct {
	cfg   pool.Config
	job   pool.Job
	block *blockchain.Block
}

func (s *stubEngine) Config() pool.Config { return s.cfg }

func (s *stubEngine) CurrentJob() pool.Job { return s.job }

func (s *stubEngine) RefreshWork(context.Context) (pool.Job, error) {
	s.job = pool.Job{
		ID:              "job-1",
		Height:          s.block.Height,
		Parent:          "parent",
		BlockDifficulty: s.block.Difficulty,
		PoolDifficulty:  1,
		PoolTarget:      pool.TargetHex(1),
		BlockReward:     10_000,
		Block:           *s.block,
	}
	return s.job, nil
}

func (s *stubEngine) SubmitShare(_ context.Context, job pool.Job, worker pool.WorkerIdentity, nonce uint64) (pool.ShareResult, pool.ShareCredit, error) {
	result, err := pool.ValidateShare(&job.Block, nonce, job.PoolDifficulty, job.BlockDifficulty)
	if err != nil {
		return pool.ShareResult{}, pool.ShareCredit{}, err
	}
	if result.Kind == pool.ShareInvalid {
		return result, pool.ShareCredit{}, fmt.Errorf("share below pool difficulty")
	}
	return result, pool.ShareCredit{Worker: worker, Value: 1}, nil
}

func TestStratumSubscribeAuthorizeSubmit(t *testing.T) {
	genesis := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(genesis, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	block.Difficulty = 1
	block.MinerAddress = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	block.UpdateMerkleRoot()

	engine := &stubEngine{
		cfg:   pool.DefaultConfig(),
		block: block,
	}
	server := stratum.NewServer(engine, func(string, ...any) {})

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
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	writeLine := func(v any) {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, '\n')
		if _, err := conn.Write(raw); err != nil {
			t.Fatal(err)
		}
	}

	writeLine(map[string]any{"id": 1, "method": "mining.subscribe", "params": []any{}})
	readResponse(t, conn, reader)

	writeLine(map[string]any{"id": 2, "method": "mining.authorize", "params": []any{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.worker", "x"}})
	readResponse(t, conn, reader)
	readNotification(t, conn, reader)

	nonce := findShareNonce(t, block, 1)
	writeLine(map[string]any{"id": 3, "method": "mining.submit", "params": []any{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.worker", "job-1", nonce}})
	readResponse(t, conn, reader)
}

func readResponse(t *testing.T, conn net.Conn, reader *bufio.Reader) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Error any `json:"error"`
	}
	if err := json.Unmarshal(line, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error != nil {
		t.Fatalf("unexpected error response: %s", line)
	}
}

func readNotification(t *testing.T, conn net.Conn, reader *bufio.Reader) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(line, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Method != "mining.notify" {
		t.Fatalf("method = %q", payload.Method)
	}
}

func findShareNonce(t *testing.T, block *blockchain.Block, difficulty uint32) uint64 {
	t.Helper()
	work := gpuminer.Work{Block: block}
	_ = work
	for nonce := uint64(0); nonce < 1_000_000; nonce++ {
		result, err := pool.ValidateShare(block, nonce, difficulty, block.Difficulty)
		if err != nil {
			t.Fatal(err)
		}
		if result.Kind != pool.ShareInvalid {
			return nonce
		}
	}
	t.Fatalf("no share nonce found")
	return 0
}
