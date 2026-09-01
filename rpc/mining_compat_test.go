package rpc

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/consensus"
)

func TestBuildPOWCompatWorkMatchesRVNAndEthFieldNames(t *testing.T) {
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	previous := blockchain.NewGenesisBlock()
	block, err := blockchain.NewBlockFromMempool(previous, mempool.NewMempool())
	if err != nil {
		t.Fatal(err)
	}
	block.Timestamp = previous.Timestamp + 1
	block.Difficulty = 1
	block.MinerAddress = address
	block.UpdateMerkleRoot()

	reward := consensus.BlockSubsidy(block.Height)
	compat := buildPOWCompatWork(block, previous.Hash(), reward)

	gbt, ok := compat.GetBlockTemplate["previousblockhash"].(string)
	if !ok || gbt != previous.Hash() {
		t.Fatalf("getblocktemplate.previousblockhash = %#v", compat.GetBlockTemplate["previousblockhash"])
	}
	if compat.GetBlockTemplate["height"] != block.Height {
		t.Fatalf("getblocktemplate.height = %#v", compat.GetBlockTemplate["height"])
	}
	if compat.GetBlockTemplate["coinbasevalue"] != reward {
		t.Fatalf("getblocktemplate.coinbasevalue = %#v", compat.GetBlockTemplate["coinbasevalue"])
	}

	if compat.EthGetWork["header_hash"] != block.Hash() {
		t.Fatalf("eth_getWork.header_hash = %#v", compat.EthGetWork["header_hash"])
	}
	if compat.EthGetWork["boundary"] != compat.GetBlockTemplate["target"] {
		t.Fatalf("eth_getWork.boundary = %#v target = %#v", compat.EthGetWork["boundary"], compat.GetBlockTemplate["target"])
	}
}
