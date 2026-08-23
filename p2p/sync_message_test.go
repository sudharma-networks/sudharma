package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestGetBlocksMessageEncodeDecode(t *testing.T) {
	data, err :=
		NewGetBlocksMessage(
			5,
			25,
		)

	if err != nil {
		t.Fatal(err)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		t.Fatal(err)
	}

	request, err :=
		DecodeGetBlocks(message)

	if err != nil {
		t.Fatal(err)
	}

	if request.StartHeight != 5 {
		t.Fatalf(
			"expected start height 5, got %d",
			request.StartHeight,
		)
	}

	if request.Limit != 25 {
		t.Fatalf(
			"expected limit 25, got %d",
			request.Limit,
		)
	}
}

func TestGetBlocksRejectsZeroLimit(t *testing.T) {
	if _, err :=
		NewGetBlocksMessage(
			1,
			0,
		); err == nil {

		t.Fatal(
			"zero block request limit was accepted",
		)
	}
}

func TestGetBlocksRejectsOversizedLimit(t *testing.T) {
	if _, err :=
		NewGetBlocksMessage(
			1,
			MaxBlocksPerMessage+1,
		); err == nil {

		t.Fatal(
			"oversized block request was accepted",
		)
	}
}

func TestBlocksMessageEncodeDecode(t *testing.T) {
	genesis :=
		blockchain.NewGenesisBlock()

	block := &blockchain.Block{
		Version:      1,
		Height:       1,
		Timestamp:    genesis.Timestamp + 60,
		PreviousHash: genesis.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	data, err :=
		NewBlocksMessage(
			[]*blockchain.Block{
				block,
			},
		)

	if err != nil {
		t.Fatal(err)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		t.Fatal(err)
	}

	blocks, err :=
		DecodeBlocks(message)

	if err != nil {
		t.Fatal(err)
	}

	if len(blocks) != 1 {
		t.Fatalf(
			"expected 1 block, got %d",
			len(blocks),
		)
	}

	if blocks[0].Height != 1 {
		t.Fatalf(
			"expected block height 1, got %d",
			blocks[0].Height,
		)
	}

	if blocks[0].Hash() != block.Hash() {
		t.Fatal(
			"decoded block hash does not match original",
		)
	}
}

func TestNilBlockInBlocksMessageRejected(t *testing.T) {
	if _, err :=
		NewBlocksMessage(
			[]*blockchain.Block{
				nil,
			},
		); err == nil {

		t.Fatal(
			"nil block was accepted",
		)
	}
}

func TestEmptyBlocksMessageRejected(t *testing.T) {
	if _, err :=
		NewBlocksMessage(
			nil,
		); err == nil {

		t.Fatal(
			"empty blocks response was accepted",
		)
	}
}

func TestNonConsecutiveBlocksRejected(t *testing.T) {
	genesis :=
		blockchain.NewGenesisBlock()

	block1 := &blockchain.Block{
		Version:      1,
		Height:       1,
		Timestamp:    genesis.Timestamp + 60,
		PreviousHash: genesis.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: nil,
	}

	block1.UpdateMerkleRoot()

	block3 := &blockchain.Block{
		Version:      1,
		Height:       3,
		Timestamp:    block1.Timestamp + 60,
		PreviousHash: block1.Hash(),
		Difficulty:   1,
		Nonce:        0,
		Transactions: nil,
	}

	block3.UpdateMerkleRoot()

	data, err :=
		NewBlocksMessage(
			[]*blockchain.Block{
				block1,
				block3,
			},
		)

	if err != nil {
		t.Fatal(err)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		t.Fatal(err)
	}

	if _, err :=
		DecodeBlocks(message); err == nil {

		t.Fatal(
			"non-consecutive blocks were accepted",
		)
	}
}
