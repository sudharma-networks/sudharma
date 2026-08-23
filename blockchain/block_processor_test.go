package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestProcessBlock(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	minerWallet, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}

	state := NewState()

	if state.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatalf(
			"expected permanent treasury %s, got %s",
			params.DevelopmentTreasuryAddress,
			state.DevelopmentAddress(),
		)
	}

	initialSenderBalance :=
		uint64(1000) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialSenderBalance,
	); err != nil {
		t.Fatal(err)
	}

	amount :=
		uint64(100) *
			params.CoinDecimals

	tx := transactions.NewTransaction(
		sender.Address,
		receiver.Address,
		amount,
		1,
	)

	if err := tx.Sign(sender); err != nil {
		t.Fatal(err)
	}

	block := &Block{
		Version:      1,
		Height:       1,
		Timestamp:    1786924860,
		PreviousHash: NewGenesisBlock().Hash(),
		Difficulty:   1,
		Nonce:        0,
		MinerAddress: minerWallet.Address,
		Transactions: []*transactions.Transaction{
			tx,
		},
	}

	block.UpdateMerkleRoot()

	reward, err :=
		ProcessBlock(
			state,
			block,
			minerWallet.Address,
		)

	if err != nil {
		t.Fatal(err)
	}

	expectedTotalFee :=
		transactions.CalculateFee(
			amount,
		)

	expectedDevFee :=
		transactions.DevelopmentFee(
			amount,
		)

	expectedMinerFee :=
		transactions.MiningFee(
			amount,
		)

	expectedSender :=
		initialSenderBalance -
			amount -
			expectedTotalFee

	expectedMinerReward :=
		consensus.BlockSubsidy(
			block.Height,
		) +
			expectedMinerFee

	if state.Balance(
		sender.Address,
	) != expectedSender {

		t.Fatalf(
			"wrong sender balance: expected %d, got %d",
			expectedSender,
			state.Balance(sender.Address),
		)
	}

	if state.Balance(
		receiver.Address,
	) != amount {

		t.Fatalf(
			"wrong receiver balance: expected %d, got %d",
			amount,
			state.Balance(receiver.Address),
		)
	}

	if state.Balance(
		params.DevelopmentTreasuryAddress,
	) != expectedDevFee {

		t.Fatalf(
			"wrong development balance: expected %d, got %d",
			expectedDevFee,
			state.Balance(
				params.DevelopmentTreasuryAddress,
			),
		)
	}

	if state.Balance(
		minerWallet.Address,
	) != expectedMinerReward {

		t.Fatalf(
			"wrong miner balance: expected %d, got %d",
			expectedMinerReward,
			state.Balance(minerWallet.Address),
		)
	}

	if reward != expectedMinerReward {
		t.Fatalf(
			"wrong returned miner reward: expected %d, got %d",
			expectedMinerReward,
			reward,
		)
	}

	if state.AccountNonce(
		sender.Address,
	) != 1 {

		t.Fatalf(
			"expected sender nonce 1, got %d",
			state.AccountNonce(sender.Address),
		)
	}

	if !state.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"transaction was not marked processed",
		)
	}

	expectedIssuedSupply :=
		consensus.BlockSubsidy(
			block.Height,
		)

	if state.IssuedSupply() !=
		expectedIssuedSupply {

		t.Fatalf(
			"wrong issued supply: expected %d, got %d",
			expectedIssuedSupply,
			state.IssuedSupply(),
		)
	}
}
