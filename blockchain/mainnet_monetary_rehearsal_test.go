package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/consensus"
	"github.com/sudharma-networks/sudharma/params"
)

// TestMainnetMonetaryRehearsalSample exercises mainnet-policy block processing
// at the first subsidy block, the final subsidy block, and the first fee-only
// block after the 51M cap is reached.
func TestMainnetMonetaryRehearsalSample(t *testing.T) {
	state := NewState()
	miner := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	finalHeight := params.MainnetFinalSubsidyHeight

	cases := []struct {
		name   string
		height uint64
	}{
		{name: "first-mainnet-subsidy", height: 1},
		{name: "final-mainnet-subsidy", height: finalHeight},
		{name: "post-cap-fee-only", height: finalHeight + 1},
	}

	var issued uint64
	for _, tc := range cases {
		block := &Block{
			Version:      1,
			Height:       tc.height,
			Timestamp:    int64(1_700_000_000 + int64(tc.height)),
			PreviousHash: "rehearsal-prev",
			Difficulty:   1,
			Nonce:        1,
			MinerAddress: miner,
		}
		block.UpdateMerkleRoot()

		reward, err := ProcessBlockFor(state, params.MonetaryPolicyMainnet, block, miner)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		wantSubsidy, err := consensus.BlockSubsidyFor(params.MonetaryPolicyMainnet, tc.height)
		if err != nil {
			t.Fatalf("%s subsidy lookup: %v", tc.name, err)
		}
		if reward != wantSubsidy {
			t.Fatalf("%s reward = %d want %d", tc.name, reward, wantSubsidy)
		}
		if tc.height > finalHeight && wantSubsidy != 0 {
			t.Fatalf("%s subsidy = %d after final height", tc.name, wantSubsidy)
		}

		issued = state.IssuedSupply()
		if issued > params.MainnetMaxSupply {
			t.Fatalf("%s issued %d exceeds cap %d", tc.name, issued, params.MainnetMaxSupply)
		}
	}

	if state.Balance(miner) == 0 {
		t.Fatal("expected miner balance after rehearsal blocks")
	}
}
