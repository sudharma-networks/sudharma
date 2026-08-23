package blockchain

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestNewStateUsesPermanentTreasury(t *testing.T) {
	state := NewState()

	if state.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatalf(
			"expected treasury %s, got %s",
			params.DevelopmentTreasuryAddress,
			state.DevelopmentAddress(),
		)
	}
}

func TestTreasuryCannotBeChanged(t *testing.T) {
	state := NewState()

	original :=
		state.DevelopmentAddress()

	state.SetDevelopmentAddress(
		"attacker-development-address",
	)

	if state.DevelopmentAddress() != original {
		t.Fatalf(
			"treasury address changed from %s to %s",
			original,
			state.DevelopmentAddress(),
		)
	}
}

func TestSettingCorrectTreasuryIsAllowed(t *testing.T) {
	state := NewState()

	state.SetDevelopmentAddress(
		params.DevelopmentTreasuryAddress,
	)

	if state.DevelopmentAddress() !=
		params.DevelopmentTreasuryAddress {

		t.Fatal(
			"correct treasury address was not preserved",
		)
	}
}
