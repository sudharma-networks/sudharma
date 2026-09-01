package stratum

import (
	"context"
	"fmt"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/gpuminer"
	"github.com/sudharma-networks/sudharma/pool"
)

func TestCommandBackendShareMinerUsesHasherNonce(t *testing.T) {
	genesis := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(genesis, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	block.Difficulty = 1
	block.MinerAddress = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	block.UpdateMerkleRoot()

	job := Job{
		ID:              "job-gpu",
		PoolDifficulty:  1,
		BlockDifficulty: 1,
		PoolTarget:      pool.TargetHex(1),
		Block:           *block,
	}

	miner := CommandBackendShareMiner{
		Backend: gpuminer.CommandBackend{
			Path: "khushi-miner-nvidia.exe",
			Run: func(context.Context, string, []string) ([]byte, error) {
				var validNonce uint64
				for nonce := uint64(0); nonce < 500_000; nonce++ {
					result, err := pool.ValidateShare(block, nonce, 1, 1)
					if err != nil {
						t.Fatal(err)
					}
					if result.Kind != pool.ShareInvalid {
						validNonce = nonce
						break
					}
				}
				return []byte(fmt.Sprintf("staging-solution-nonce=%d\n", validNonce)), nil
			},
		},
	}
	nonce, result, err := miner.MineShare(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind == pool.ShareInvalid {
		t.Fatalf("unexpected invalid share for nonce %d", nonce)
	}
}

func TestWorkFromCandidateBlockBuildsHeaderPrefix(t *testing.T) {
	genesis := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(genesis, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	block.UpdateMerkleRoot()
	work := gpuminer.WorkFromCandidateBlock(block, 1, "ffff")
	if work.HeaderPrefix == "" || work.Target != "ffff" {
		t.Fatalf("work = %+v", work)
	}
}
