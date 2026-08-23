package miner

import (
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/blockchain/mempool"
	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestMineNextBlockPipeline(t *testing.T) {
	// --------------------------------------------
	// Wallets
	// --------------------------------------------

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

	// --------------------------------------------
	// Chain + state + mempool
	// --------------------------------------------

	chain :=
		blockchain.NewChain()

	state :=
		blockchain.NewState()

	if state.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatalf(
			"expected permanent treasury %s, got %s",
			params.DevelopmentTreasuryAddress,
			state.DevelopmentAddress(),
		)
	}

	pool :=
		mempool.NewMempool()

	initialBalance :=
		uint64(100) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialBalance,
	); err != nil {
		t.Fatal(err)
	}

	// --------------------------------------------
	// Transaction
	// --------------------------------------------

	amount :=
		uint64(10) *
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

	if err :=
		blockchain.ValidateMempoolTransaction(
			state,
			pool.AllTransactions(),
			tx,
		); err != nil {

		t.Fatal(err)
	}

	if err := pool.AddTransaction(
		tx,
	); err != nil {
		t.Fatal(err)
	}

	if pool.Count() != 1 {
		t.Fatalf(
			"expected mempool count 1, got %d",
			pool.Count(),
		)
	}

	// --------------------------------------------
	// Mine complete block
	// --------------------------------------------

	result, reward, err :=
		MineNextBlock(
			chain,
			state,
			pool,
			minerWallet.Address,
			1_000_000,
		)

	if err != nil {
		t.Fatalf(
			"mining pipeline failed: %v",
			err,
		)
	}

	if !result.Found {
		t.Fatal(
			"mining pipeline did not find a block",
		)
	}

	if result.Block == nil {
		t.Fatal(
			"mining pipeline returned nil block",
		)
	}

	if result.Block.MinerAddress !=
		minerWallet.Address {

		t.Fatalf(
			"wrong miner address in block: expected %s, got %s",
			minerWallet.Address,
			result.Block.MinerAddress,
		)
	}

	// --------------------------------------------
	// Chain verification
	// --------------------------------------------

	if chain.Height() != 1 {
		t.Fatalf(
			"expected chain height 1, got %d",
			chain.Height(),
		)
	}

	if chain.Length() != 2 {
		t.Fatalf(
			"expected chain length 2, got %d",
			chain.Length(),
		)
	}

	if chain.Tip().Hash() !=
		result.Block.Hash() {

		t.Fatal(
			"chain tip does not match mined block",
		)
	}

	// --------------------------------------------
	// Mempool verification
	// --------------------------------------------

	if pool.Count() != 0 {
		t.Fatalf(
			"expected empty mempool, got %d",
			pool.Count(),
		)
	}

	// --------------------------------------------
	// Balance verification
	// --------------------------------------------

	totalFee :=
		transactions.CalculateFee(
			amount,
		)

	devFee :=
		transactions.DevelopmentFee(
			amount,
		)

	minerFee :=
		transactions.MiningFee(
			amount,
		)

	expectedSender :=
		initialBalance -
			amount -
			totalFee

	expectedMiner :=
		consensus.BlockSubsidy(1) +
			minerFee

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
	) != devFee {

		t.Fatalf(
			"wrong development balance: expected %d, got %d",
			devFee,
			state.Balance(
				params.DevelopmentTreasuryAddress,
			),
		)
	}

	if state.Balance(
		minerWallet.Address,
	) != expectedMiner {

		t.Fatalf(
			"wrong miner balance: expected %d, got %d",
			expectedMiner,
			state.Balance(minerWallet.Address),
		)
	}

	if reward != expectedMiner {
		t.Fatalf(
			"wrong returned reward: expected %d, got %d",
			expectedMiner,
			reward,
		)
	}

	// --------------------------------------------
	// Confirmation verification
	// --------------------------------------------

	if !state.IsTransactionProcessed(
		tx.ID,
	) {
		t.Fatal(
			"transaction was not marked confirmed",
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

	expectedSupply :=
		consensus.BlockSubsidy(1)

	if state.IssuedSupply() !=
		expectedSupply {

		t.Fatalf(
			"wrong issued supply: expected %d, got %d",
			expectedSupply,
			state.IssuedSupply(),
		)
	}
}

func TestMineNextBlockRejectsNilInputs(t *testing.T) {
	chain :=
		blockchain.NewChain()

	state :=
		blockchain.NewState()

	pool :=
		mempool.NewMempool()

	minerWallet, err :=
		wallet.NewWallet()

	if err != nil {
		t.Fatal(err)
	}

	if _, _, err :=
		MineNextBlock(
			nil,
			state,
			pool,
			minerWallet.Address,
			100,
		); err == nil {

		t.Fatal(
			"nil chain was accepted",
		)
	}

	if _, _, err :=
		MineNextBlock(
			chain,
			nil,
			pool,
			minerWallet.Address,
			100,
		); err == nil {

		t.Fatal(
			"nil state was accepted",
		)
	}

	if _, _, err :=
		MineNextBlock(
			chain,
			state,
			nil,
			minerWallet.Address,
			100,
		); err == nil {

		t.Fatal(
			"nil mempool was accepted",
		)
	}
}
