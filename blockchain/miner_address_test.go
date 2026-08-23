package blockchain

import "testing"

func TestMinerAddressChangesBlockHash(t *testing.T) {
	previous :=
		NewGenesisBlock()

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    previous.Timestamp + 60,
		PreviousHash: previous.Hash(),
		Difficulty:   1,
		Nonce:        123,
		MinerAddress: "miner-a",
		Transactions: nil,
	}

	block.UpdateMerkleRoot()

	hashA :=
		block.Hash()

	block.MinerAddress =
		"attacker-wallet"

	hashB :=
		block.Hash()

	if hashA == hashB {
		t.Fatal(
			"changing miner address did not change block hash",
		)
	}
}
