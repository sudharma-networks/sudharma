package stratum

import (
	"bufio"
	"context"
	"net"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
)

func TestParseNotifyBuildsMineableBlock(t *testing.T) {
	params := []any{
		"job-1", "1", "parent", "merkle", "1", "100", "abc", "1700000000", "0",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true,
	}
	job, err := ParseNotify(params)
	if err != nil {
		t.Fatal(err)
	}
	if job.Block.Height != 1 || job.PoolDifficulty != 1 || job.BlockDifficulty != 100 {
		t.Fatalf("job = %+v", job)
	}
	if job.Block.MinerAddress != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("miner = %q", job.Block.MinerAddress)
	}
}

func TestParsePoolURL(t *testing.T) {
	host, err := ParsePoolURL("stratum+tcp://127.0.0.1:3333")
	if err != nil {
		t.Fatal(err)
	}
	if host != "127.0.0.1:3333" {
		t.Fatalf("host = %q", host)
	}
}

func TestWorkerLogin(t *testing.T) {
	login, err := WorkerLogin("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "rig1")
	if err != nil {
		t.Fatal(err)
	}
	if login != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.rig1" {
		t.Fatalf("login = %q", login)
	}
}

func TestStratumClientHandshakeAndShareSubmit(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		reader := bufio.NewReader(serverConn)
		method, _, err := ReadRequest(reader)
		if err != nil || method != "mining.subscribe" {
			t.Errorf("subscribe: %v", err)
			return
		}
		_ = WriteResponse(serverConn, 1, []any{[]any{"mining.notify", "abc"}, "abc", 4})
		method, _, err = ReadRequest(reader)
		if err != nil || method != "mining.authorize" {
			t.Errorf("authorize: %v", err)
			return
		}
		_ = WriteResponse(serverConn, 2, true)

		genesis := blockchain.NewGenesisBlock()
		block, err := blockchain.NewBlockFromMempool(genesis, mempool.NewMempool())
		if err != nil {
			t.Error(err)
			return
		}
		block.Difficulty = 1
		block.MinerAddress = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		block.UpdateMerkleRoot()
		_ = WriteNotification(serverConn, "mining.notify",
			"job-1", "1", block.PreviousHash, block.MerkleRoot, "1", "1", "ffff", block.Timestamp, block.Version, block.MinerAddress, true)

		method, _, err = ReadRequest(reader)
		if err != nil || method != "mining.submit" {
			t.Errorf("submit: %v", err)
			return
		}
		_ = WriteResponse(serverConn, 3, true)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := ServeConn(ctx, clientConn, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.worker", "x")
	if err != nil {
		t.Fatal(err)
	}

	loop := &Loop{
		Client: client,
		Once:   true,
	}
	shares, blocks, err := loop.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if shares+blocks < 1 {
		t.Fatalf("shares=%d blocks=%d", shares, blocks)
	}
}
