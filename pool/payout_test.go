package pool

import "testing"

func TestPPSLedgerCreditsEachShare(t *testing.T) {
	ledger, err := NewPayoutLedger(SchemePPS, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	worker, _ := ParseWorkerIdentity("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.worker")
	share := ShareResult{
		Kind:            ShareValid,
		PoolDifficulty:  1,
		BlockDifficulty: 100,
		ShareWork:       1,
	}
	credit := ledger.CreditShare(10_000, share, worker, "job1", 1)
	if credit.Value == 0 {
		t.Fatal("expected non-zero share value")
	}
	if ledger.Balance(worker.Address) != credit.Value {
		t.Fatalf("balance = %d credit = %d", ledger.Balance(worker.Address), credit.Value)
	}
}

func TestPPLNSLedgerDistributesBlockReward(t *testing.T) {
	ledger, err := NewPayoutLedger(SchemePPLNS, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := ParseWorkerIdentity("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.a")
	b, _ := ParseWorkerIdentity("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.b")

	shareA := ShareResult{Kind: ShareValid, PoolDifficulty: 1, BlockDifficulty: 100, ShareWork: 1}
	shareB := ShareResult{Kind: ShareValid, PoolDifficulty: 1, BlockDifficulty: 100, ShareWork: 3}
	ledger.CreditShare(10_000, shareA, a, "job1", 1)
	ledger.CreditShare(10_000, shareB, b, "job2", 1)

	blockShare := ShareResult{Kind: ShareBlock, PoolDifficulty: 1, BlockDifficulty: 100, ShareWork: 1}
	ledger.CreditShare(10_000, blockShare, a, "job3", 1)

	if ledger.Balance(a.Address) == 0 || ledger.Balance(b.Address) == 0 {
		t.Fatalf("balances a=%d b=%d", ledger.Balance(a.Address), ledger.Balance(b.Address))
	}
	if ledger.Balance(b.Address) <= ledger.Balance(a.Address) {
		t.Fatalf("expected worker b to earn more than worker a: a=%d b=%d", ledger.Balance(a.Address), ledger.Balance(b.Address))
	}
}

func TestSoloLedgerCreditsOnlyBlocks(t *testing.T) {
	ledger, err := NewPayoutLedger(SchemeSolo, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	worker, _ := ParseWorkerIdentity("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	share := ShareResult{Kind: ShareValid, PoolDifficulty: 1, BlockDifficulty: 100, ShareWork: 1}
	ledger.CreditShare(10_000, share, worker, "job1", 1)
	if ledger.Balance(worker.Address) != 0 {
		t.Fatalf("solo share should not credit balance, got %d", ledger.Balance(worker.Address))
	}
	blockShare := ShareResult{Kind: ShareBlock, PoolDifficulty: 1, BlockDifficulty: 100, ShareWork: 1}
	ledger.CreditShare(10_000, blockShare, worker, "job2", 1)
	if ledger.Balance(worker.Address) != 10_000 {
		t.Fatalf("balance = %d", ledger.Balance(worker.Address))
	}
}

func TestNormalizePayoutScheme(t *testing.T) {
	scheme, err := NormalizePayoutScheme("fpps")
	if err != nil {
		t.Fatal(err)
	}
	if scheme != SchemeFPPS {
		t.Fatalf("scheme = %q", scheme)
	}
}
