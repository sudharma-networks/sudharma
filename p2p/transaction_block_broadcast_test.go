package p2p

import (
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

func TestTransactionBlockBroadcastBetweenNodes(t *testing.T) {
	sender, err := wallet.NewWallet()
	if err != nil { t.Fatal(err) }
	receiver, err := wallet.NewWallet()
	if err != nil { t.Fatal(err) }
	minerWallet, err := wallet.NewWallet()
	if err != nil { t.Fatal(err) }

	chainA := blockchain.NewChain()
	chainB := blockchain.NewChain()
	stateA := blockchain.NewState()
	stateB := blockchain.NewState()
	if stateA.DevelopmentAddress() != params.DevelopmentTreasuryAddress { t.Fatalf("node A has wrong treasury address: %s", stateA.DevelopmentAddress()) }
	if stateB.DevelopmentAddress() != params.DevelopmentTreasuryAddress { t.Fatalf("node B has wrong treasury address: %s", stateB.DevelopmentAddress()) }

	initialBalance := uint64(100) * params.CoinDecimals
	if err := stateA.Credit(sender.Address, initialBalance); err != nil { t.Fatal(err) }
	if err := stateB.Credit(sender.Address, initialBalance); err != nil { t.Fatal(err) }

	nodeA, err := NewNode("tx-block-node-a", "127.0.0.1:0", chainA.Height(), chainA.Tip().Hash())
	if err != nil { t.Fatal(err) }
	nodeB, err := NewNode("tx-block-node-b", "127.0.0.1:0", chainB.Height(), chainB.Tip().Hash())
	if err != nil { t.Fatal(err) }
	if err := nodeA.SetChain(chainA); err != nil { t.Fatal(err) }
	if err := nodeB.SetChain(chainB); err != nil { t.Fatal(err) }
	if err := nodeA.SetState(stateA); err != nil { t.Fatal(err) }
	if err := nodeB.SetState(stateB); err != nil { t.Fatal(err) }
	if err := nodeA.Start(); err != nil { t.Fatal(err) }
	defer nodeA.Stop()
	if err := nodeB.Start(); err != nil { t.Fatal(err) }
	defer nodeB.Stop()
	if _, err := nodeB.Connect(nodeA.ListenAddress); err != nil { t.Fatal(err) }

	amount := uint64(10) * params.CoinDecimals
	tx := transactions.NewTransaction(sender.Address, receiver.Address, amount, 1)
	if err := tx.Sign(sender); err != nil { t.Fatal(err) }
	if !tx.Verify() { t.Fatal("transaction signature verification failed") }
	poolA := nodeA.Mempool()
	if err := blockchain.ValidateMempoolTransaction(stateA, poolA.AllTransactions(), tx); err != nil { t.Fatalf("transaction failed mempool validation: %v", err) }
	if err := poolA.AddTransaction(tx); err != nil { t.Fatal(err) }
	if poolA.Count() != 1 { t.Fatalf("expected node A mempool count 1, got %d", poolA.Count()) }

	result, rewardA, err := miner.MineNextBlock(chainA, stateA, poolA, minerWallet.Address, 1_000_000)
	if err != nil { t.Fatalf("node A mining failed: %v", err) }
	if !result.Found { t.Fatal("node A failed to mine block") }
	if result.Block == nil { t.Fatal("mining result block is nil") }
	if result.Block.MinerAddress != minerWallet.Address { t.Fatalf("wrong miner address in mined block: expected %s, got %s", minerWallet.Address, result.Block.MinerAddress) }
	nodeA.RefreshChainStatus()
	if chainA.Height() != 1 { t.Fatalf("expected node A height 1, got %d", chainA.Height()) }
	if poolA.Count() != 0 { t.Fatalf("expected node A mempool empty after mining, got %d", poolA.Count()) }

	if err := nodeA.BroadcastBlock(result.Block); err != nil { t.Fatalf("block broadcast failed: %v", err) }
	deadline := time.Now().Add(2 * time.Second)
	for chainB.Height() == 0 && time.Now().Before(deadline) { time.Sleep(10 * time.Millisecond) }
	if chainB.Height() != 1 { t.Fatalf("expected node B height 1, got %d", chainB.Height()) }
	if chainB.Tip().Hash() != chainA.Tip().Hash() { t.Fatal("node B tip hash does not match node A") }
	height, _ := nodeB.AdvertisedChainStatus()
	if height != 1 { t.Fatalf("expected node B advertised height 1, got %d", height) }

	addresses := []string{sender.Address, receiver.Address, params.DevelopmentTreasuryAddress, minerWallet.Address}
	for _, address := range addresses {
		balanceA := stateA.Balance(address)
		balanceB := stateB.Balance(address)
		if balanceA != balanceB { t.Fatalf("state mismatch for %s: node A %d, node B %d", address, balanceA, balanceB) }
	}
	if stateA.AccountNonce(sender.Address) != stateB.AccountNonce(sender.Address) { t.Fatal("sender nonce differs between nodes") }
	if stateA.AccountNonce(sender.Address) != 1 { t.Fatalf("expected sender nonce 1, got %d", stateA.AccountNonce(sender.Address)) }
	if !stateA.IsTransactionProcessed(tx.ID) { t.Fatal("node A did not mark transaction confirmed") }
	if !stateB.IsTransactionProcessed(tx.ID) { t.Fatal("node B did not mark transaction confirmed") }
	if stateA.IssuedSupply() != stateB.IssuedSupply() { t.Fatalf("issued supply mismatch: node A %d, node B %d", stateA.IssuedSupply(), stateB.IssuedSupply()) }

	expectedMinerReward := uint64(50)*params.CoinDecimals + transactions.MiningFee(amount)
	if rewardA != expectedMinerReward { t.Fatalf("wrong node A reward: expected %d, got %d", expectedMinerReward, rewardA) }
	if stateA.Balance(minerWallet.Address) != expectedMinerReward { t.Fatalf("node A calculated wrong miner reward: expected %d, got %d", expectedMinerReward, stateA.Balance(minerWallet.Address)) }
	if stateB.Balance(minerWallet.Address) != expectedMinerReward { t.Fatalf("node B calculated wrong miner reward: expected %d, got %d", expectedMinerReward, stateB.Balance(minerWallet.Address)) }

	totalFee := transactions.CalculateFee(amount)
	expectedSender := initialBalance - amount - totalFee
	if stateB.Balance(sender.Address) != expectedSender { t.Fatalf("wrong sender balance on node B: expected %d, got %d", expectedSender, stateB.Balance(sender.Address)) }
	if stateB.Balance(receiver.Address) != amount { t.Fatalf("wrong receiver balance on node B: expected %d, got %d", amount, stateB.Balance(receiver.Address)) }
	expectedDevelopmentFee := transactions.DevelopmentFee(amount)
	if stateA.Balance(params.DevelopmentTreasuryAddress) != expectedDevelopmentFee { t.Fatalf("wrong development balance on node A: expected %d, got %d", expectedDevelopmentFee, stateA.Balance(params.DevelopmentTreasuryAddress)) }
	if stateB.Balance(params.DevelopmentTreasuryAddress) != expectedDevelopmentFee { t.Fatalf("wrong development balance on node B: expected %d, got %d", expectedDevelopmentFee, stateB.Balance(params.DevelopmentTreasuryAddress)) }
}
