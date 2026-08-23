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

func TestMineValidMempoolTransaction(t *testing.T) {
	// ------------------------------------------------
	// Create wallets
	// ------------------------------------------------

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

	// ------------------------------------------------
	// Create blockchain state
	// ------------------------------------------------

	state := blockchain.NewState()

	if state.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatalf(
			"expected permanent treasury %s, got %s",
			params.DevelopmentTreasuryAddress,
			state.DevelopmentAddress(),
		)
	}

	// Development-test funding.
	//
	// This is only a test fixture.
	// It is NOT a mainnet premine.
	initialSenderBalance :=
		uint64(100) *
			params.CoinDecimals

	if err := state.Credit(
		sender.Address,
		initialSenderBalance,
	); err != nil {
		t.Fatal(err)
	}

	// ------------------------------------------------
	// Create transaction
	// ------------------------------------------------

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

	if !tx.Verify() {
		t.Fatal(
			"signed transaction verification failed",
		)
	}

	// ------------------------------------------------
	// Validate before mempool admission
	// ------------------------------------------------

	pool := mempool.NewMempool()

	if err :=
		blockchain.ValidateMempoolTransaction(
			state,
			pool.AllTransactions(),
			tx,
		); err != nil {

		t.Fatalf(
			"valid transaction rejected by mempool validator: %v",
			err,
		)
	}

	if err := pool.AddTransaction(tx); err != nil {
		t.Fatal(err)
	}

	if pool.Count() != 1 {
		t.Fatalf(
			"expected mempool count 1, got %d",
			pool.Count(),
		)
	}

	// ------------------------------------------------
	// Create blockchain
	// ------------------------------------------------

	chain := blockchain.NewChain()

	previous := chain.Tip()

	if previous == nil {
		t.Fatal(
			"chain tip cannot be nil",
		)
	}

	// ------------------------------------------------
	// Build candidate block from mempool
	// ------------------------------------------------

	block, err :=
		blockchain.NewBlockFromMempool(
			previous,
			pool,
		)

	if err != nil {
		t.Fatal(err)
	}

	if block.Height != 1 {
		t.Fatalf(
			"expected candidate height 1, got %d",
			block.Height,
		)
	}

	if len(block.Transactions) != 1 {
		t.Fatalf(
			"expected 1 block transaction, got %d",
			len(block.Transactions),
		)
	}

	// Miner address is part of the block header and therefore
	// must be set before Proof-of-Work is performed.
	block.MinerAddress = minerWallet.Address

	// Ensure candidate uses the network-required
	// difficulty for its actual timestamp.
	actualBlockTime :=
		block.Timestamp -
			previous.Timestamp

	block.Difficulty =
		consensus.NextDifficulty(
			previous.Difficulty,
			actualBlockTime,
		)

	block.UpdateMerkleRoot()

	// ------------------------------------------------
	// Mine Proof-of-Work
	// ------------------------------------------------

	result := Mine(
		block,
		0,
		1_000_000,
	)

	if !result.Found {
		t.Fatal(
			"failed to mine candidate block",
		)
	}

	if result.Block != block {
		t.Fatal(
			"mining result returned wrong block",
		)
	}

	if block.Nonce != result.Nonce {
		t.Fatal(
			"block nonce does not match mining result",
		)
	}

	// ------------------------------------------------
	// Validate mined block
	// ------------------------------------------------

	if err :=
		blockchain.ValidateBlockBasic(
			block,
			previous,
		); err != nil {

		t.Fatalf(
			"mined block failed validation: %v",
			err,
		)
	}

	// ------------------------------------------------
	// Process block state changes
	// ------------------------------------------------

	minerReward, err :=
		blockchain.ProcessBlock(
			state,
			block,
			minerWallet.Address,
		)

	if err != nil {
		t.Fatalf(
			"failed to process mined block: %v",
			err,
		)
	}

	// ------------------------------------------------
	// Add block to chain
	// ------------------------------------------------

	if err := chain.AddBlock(block); err != nil {
		t.Fatalf(
			"failed to add mined block to chain: %v",
			err,
		)
	}

	if chain.Height() != 1 {
		t.Fatalf(
			"expected chain height 1, got %d",
			chain.Height(),
		)
	}

	if chain.Tip().Hash() != block.Hash() {
		t.Fatal(
			"chain tip does not match mined block",
		)
	}

	// ------------------------------------------------
	// Remove confirmed transaction from mempool
	// ------------------------------------------------

	for _, confirmedTx := range block.Transactions {
		pool.RemoveTransaction(
			confirmedTx.ID,
		)
	}

	if pool.Count() != 0 {
		t.Fatalf(
			"expected empty mempool after confirmation, got %d",
			pool.Count(),
		)
	}

	// ------------------------------------------------
	// Verify transaction accounting
	// ------------------------------------------------

	totalFee :=
		transactions.CalculateFee(amount)

	developmentFee :=
		transactions.DevelopmentFee(amount)

	minerFee :=
		transactions.MiningFee(amount)

	expectedSender :=
		initialSenderBalance -
			amount -
			totalFee

	expectedReceiver :=
		amount

	expectedDevelopment :=
		developmentFee

	expectedMinerReward :=
		consensus.BlockSubsidy(
			block.Height,
		) +
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
	) != expectedReceiver {

		t.Fatalf(
			"wrong receiver balance: expected %d, got %d",
			expectedReceiver,
			state.Balance(receiver.Address),
		)
	}

	if state.Balance(
		params.DevelopmentTreasuryAddress,
	) != expectedDevelopment {

		t.Fatalf(
			"wrong development balance: expected %d, got %d",
			expectedDevelopment,
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

	if minerReward != expectedMinerReward {
		t.Fatalf(
			"wrong returned miner reward: expected %d, got %d",
			expectedMinerReward,
			minerReward,
		)
	}

	// ------------------------------------------------
	// Verify nonce/replay state
	// ------------------------------------------------

	if state.AccountNonce(
		sender.Address,
	) != 1 {

		t.Fatalf(
			"expected sender nonce 1, got %d",
			state.AccountNonce(sender.Address),
		)
	}

	if !state.IsTransactionProcessed(tx.ID) {
		t.Fatal(
			"confirmed transaction was not marked processed",
		)
	}

	// ------------------------------------------------
	// Verify SUDH issuance
	// ------------------------------------------------

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
