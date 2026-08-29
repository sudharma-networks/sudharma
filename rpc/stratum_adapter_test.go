package rpc

import (
	"context"
	"errors"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/pool/stratum"
)

type testBlockProvider struct {
	block *blockchain.Block
	err   error
	seen  int
}

func (p *testBlockProvider) provider() MiningBlockProvider {
	return func() (*blockchain.Block, error) {
		p.seen++
		return p.block, p.err
	}
}

func TestStratumWorkSourceRejectsNilDependencies(t *testing.T) {
	provider := &testBlockProvider{}
	if _, err := NewStratumWorkSource(nil, provider.provider()); err == nil {
		t.Fatal("expected nil mining service to fail")
	}
	if _, err := NewStratumWorkSource(NewMiningWorkService(nil), nil); err == nil {
		t.Fatal("expected nil block provider to fail")
	}
}

func TestStratumWorkSourceCurrentWorkCopiesBlockAndMapsTemplateExactly(t *testing.T) {
	original := &blockchain.Block{
		Version:      2,
		Height:       7500,
		Timestamp:    1786924860,
		PreviousHash: "0123456789abcdef",
		MerkleRoot:   "fedcba9876543210",
		Difficulty:   7,
		MinerAddress: "provider-placeholder",
	}
	provider := &testBlockProvider{block: original}
	service := NewMiningWorkService(func(*blockchain.Block, uint64) bool { return true })
	adapter, err := NewStratumWorkSource(service, provider.provider())
	if err != nil {
		t.Fatal(err)
	}

	const reward = "9ccdc094489874bed888ffe4bdf9b8298f4c5131"
	got, err := adapter.CurrentWork(context.Background(), reward)
	if err != nil {
		t.Fatal(err)
	}
	if provider.seen != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.seen)
	}
	if original.MinerAddress != "provider-placeholder" {
		t.Fatalf("provider block mutated: miner address = %q", original.MinerAddress)
	}

	expectedBlock := *original
	expectedBlock.MinerAddress = reward
	expectedTemplate, err := NewMiningWorkTemplate(&expectedBlock)
	if err != nil {
		t.Fatal(err)
	}
	expected := stratum.Work{
		WorkID:          expectedTemplate.WorkID,
		Algorithm:       expectedTemplate.Algorithm,
		Version:         expectedTemplate.Version,
		Height:          expectedTemplate.Height,
		Difficulty:      expectedTemplate.Difficulty,
		TargetHex:       expectedTemplate.TargetHex,
		HeaderPrefixHex: expectedTemplate.HeaderPrefixHex,
		RewardAddress:   expectedTemplate.RewardAddress,
	}
	if got != expected {
		t.Fatalf("mapped work = %+v, want %+v", got, expected)
	}
}

func TestStratumWorkSourceCurrentWorkFailsClosedOnProviderErrors(t *testing.T) {
	service := NewMiningWorkService(nil)

	provider := &testBlockProvider{err: errors.New("provider unavailable")}
	adapter, err := NewStratumWorkSource(service, provider.provider())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.CurrentWork(context.Background(), "miner"); err == nil {
		t.Fatal("expected provider error")
	}

	provider = &testBlockProvider{}
	adapter, err = NewStratumWorkSource(service, provider.provider())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.CurrentWork(context.Background(), "miner"); err == nil {
		t.Fatal("expected nil provider block to fail")
	}
}

func TestStratumWorkSourceSubmitMapsMiningStatuses(t *testing.T) {
	tests := []struct {
		name      string
		verifier  MiningSolutionVerifier
		prepare   func(*MiningWorkService, MiningWorkTemplate)
		want      stratum.SourceResult
		wantError bool
	}{
		{
			name:     "accepted",
			verifier: func(*blockchain.Block, uint64) bool { return true },
			want:     stratum.SourceAccepted,
		},
		{
			name:     "invalid",
			verifier: func(*blockchain.Block, uint64) bool { return false },
			want:     stratum.SourceInvalid,
		},
		{
			name:     "stale",
			verifier: func(*blockchain.Block, uint64) bool { return true },
			prepare: func(service *MiningWorkService, _ MiningWorkTemplate) {
				if _, err := service.Issue(&blockchain.Block{Version: 2, Height: 7501, Difficulty: 1, MinerAddress: "replacement"}); err != nil {
					panic(err)
				}
			},
			want: stratum.SourceStale,
		},
		{
			name:     "mutated",
			verifier: func(*blockchain.Block, uint64) bool { return true },
			prepare: func(service *MiningWorkService, work MiningWorkTemplate) {
				service.mu.Lock()
				service.active.template = work
				service.active.template.Algorithm = "tampered-after-issue"
				service.mu.Unlock()
			},
			want: stratum.SourceMutated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMiningWorkService(tt.verifier)
			provider := &testBlockProvider{block: &blockchain.Block{Version: 2, Height: 7500, Difficulty: 1}}
			adapter, err := NewStratumWorkSource(service, provider.provider())
			if err != nil {
				t.Fatal(err)
			}
			work, err := adapter.CurrentWork(context.Background(), "9ccdc094489874bed888ffe4bdf9b8298f4c5131")
			if err != nil {
				t.Fatal(err)
			}
			if tt.prepare != nil {
				service.mu.RLock()
				template := service.active.template
				service.mu.RUnlock()
				tt.prepare(service, template)
			}
			got, err := adapter.Submit(context.Background(), stratum.Candidate{Work: work, Nonce: 42})
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("source result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStratumWorkSourceSubmitUsesOnlyStoredTemplateAndNonce(t *testing.T) {
	const reward = "9ccdc094489874bed888ffe4bdf9b8298f4c5131"
	service := NewMiningWorkService(func(block *blockchain.Block, nonce uint64) bool {
		return block.Height == 7500 && block.MinerAddress == reward && nonce == 42
	})
	provider := &testBlockProvider{block: &blockchain.Block{Version: 2, Height: 7500, Difficulty: 1}}
	adapter, err := NewStratumWorkSource(service, provider.provider())
	if err != nil {
		t.Fatal(err)
	}
	work, err := adapter.CurrentWork(context.Background(), reward)
	if err != nil {
		t.Fatal(err)
	}

	candidate := stratum.Candidate{
		Work:     work,
		JobID:    "attacker-job-id",
		Identity: stratum.WorkerIdentity{Wallet: "attacker", Worker: "attacker"},
		Lane:     0xffffffff,
		Nonce:    42,
	}
	candidate.Work.Algorithm = "attacker-algorithm"
	candidate.Work.Version = 99
	candidate.Work.Height = 1
	candidate.Work.Difficulty = 999
	candidate.Work.TargetHex = "attacker-target"
	candidate.Work.HeaderPrefixHex = "attacker-prefix"
	candidate.Work.RewardAddress = "attacker-reward"

	got, err := adapter.Submit(context.Background(), candidate)
	if err != nil {
		t.Fatal(err)
	}
	if got != stratum.SourceAccepted {
		t.Fatalf("source result = %q, want accepted", got)
	}
}

func TestStratumWorkSourceReplacesBoundedCurrentSnapshot(t *testing.T) {
	service := NewMiningWorkService(func(*blockchain.Block, uint64) bool { return true })
	provider := &testBlockProvider{block: &blockchain.Block{Version: 2, Height: 7500, Difficulty: 1}}
	adapter, err := NewStratumWorkSource(service, provider.provider())
	if err != nil {
		t.Fatal(err)
	}
	oldWork, err := adapter.CurrentWork(context.Background(), "9ccdc094489874bed888ffe4bdf9b8298f4c5131")
	if err != nil {
		t.Fatal(err)
	}
	provider.block = &blockchain.Block{Version: 2, Height: 7501, Difficulty: 1}
	if _, err := adapter.CurrentWork(context.Background(), "9ccdc094489874bed888ffe4bdf9b8298f4c5131"); err != nil {
		t.Fatal(err)
	}
	got, err := adapter.Submit(context.Background(), stratum.Candidate{Work: oldWork, Nonce: 42})
	if err != nil {
		t.Fatal(err)
	}
	if got != stratum.SourceStale {
		t.Fatalf("old snapshot result = %q, want stale", got)
	}
}
