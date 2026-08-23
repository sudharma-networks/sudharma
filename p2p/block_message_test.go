package p2p

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
)

func TestBlockMessageEncodeDecode(t *testing.T) {
	previous :=
		blockchain.NewGenesisBlock()

	block := &blockchain.Block{
		Version:      1,
		Height:       1,
		Timestamp:    previous.Timestamp + 60,
		PreviousHash: previous.Hash(),
		Difficulty:   previous.Difficulty,
		Nonce:        12345,
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	data, err :=
		NewBlockMessage(block)

	if err != nil {
		t.Fatalf(
			"failed to encode block: %v",
			err,
		)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		t.Fatalf(
			"failed to decode P2P message: %v",
			err,
		)
	}

	if message.Type != MessageBlock {
		t.Fatalf(
			"expected block message, got %s",
			message.Type,
		)
	}

	decoded, err :=
		DecodeBlock(message)

	if err != nil {
		t.Fatalf(
			"failed to decode block: %v",
			err,
		)
	}

	if decoded.Height != block.Height {
		t.Fatalf(
			"wrong height: expected %d, got %d",
			block.Height,
			decoded.Height,
		)
	}

	if decoded.PreviousHash !=
		block.PreviousHash {

		t.Fatal(
			"previous hash changed during transport",
		)
	}

	if decoded.MerkleRoot !=
		block.MerkleRoot {

		t.Fatal(
			"Merkle root changed during transport",
		)
	}

	if decoded.Nonce != block.Nonce {
		t.Fatalf(
			"wrong nonce: expected %d, got %d",
			block.Nonce,
			decoded.Nonce,
		)
	}

	if decoded.Hash() != block.Hash() {
		t.Fatal(
			"block hash changed during transport",
		)
	}
}

func TestNilBlockMessageRejected(t *testing.T) {
	if _, err :=
		NewBlockMessage(nil); err == nil {

		t.Fatal(
			"nil block was accepted",
		)
	}
}

func TestBlockCannotDecodeAsTransaction(t *testing.T) {
	block :=
		blockchain.NewGenesisBlock()

	data, err :=
		NewBlockMessage(block)

	if err != nil {
		t.Fatal(err)
	}

	message, err :=
		DecodeMessage(data)

	if err != nil {
		t.Fatal(err)
	}

	if _, err :=
		DecodeTransaction(message); err == nil {

		t.Fatal(
			"block decoded as transaction",
		)
	}
}
