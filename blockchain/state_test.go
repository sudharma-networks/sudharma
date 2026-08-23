package blockchain

import "testing"

func TestStateCreditDebit(t *testing.T) {
	state := NewState()

	address := "test-address"

	state.Credit(address, 1000)

	if state.Balance(address) != 1000 {
		t.Fatalf(
			"expected balance 1000, got %d",
			state.Balance(address),
		)
	}

	if err := state.Debit(address, 300); err != nil {
		t.Fatal(err)
	}

	if state.Balance(address) != 700 {
		t.Fatalf(
			"expected balance 700, got %d",
			state.Balance(address),
		)
	}
}

func TestStateInsufficientBalance(t *testing.T) {
	state := NewState()

	address := "test-address"

	state.Credit(address, 100)

	if err := state.Debit(address, 101); err == nil {
		t.Fatal("expected insufficient balance error")
	}

	if state.Balance(address) != 100 {
		t.Fatal("balance changed after failed debit")
	}
}

func TestStateTransfer(t *testing.T) {
	state := NewState()

	alice := "alice"
	bob := "bob"

	state.Credit(alice, 1000)

	if err := state.Transfer(alice, bob, 400); err != nil {
		t.Fatal(err)
	}

	if state.Balance(alice) != 600 {
		t.Fatalf(
			"expected Alice balance 600, got %d",
			state.Balance(alice),
		)
	}

	if state.Balance(bob) != 400 {
		t.Fatalf(
			"expected Bob balance 400, got %d",
			state.Balance(bob),
		)
	}
}
