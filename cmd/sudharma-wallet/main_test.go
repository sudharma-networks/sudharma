package main

import (
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestParseCoinAmount(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
	}{
		{"1", params.CoinDecimals},
		{"1.5", params.CoinDecimals + params.CoinDecimals/2},
		{"0.00000001", 1},
		{"10.00000000", 10 * params.CoinDecimals},
	}
	for _, tc := range tests {
		got, err := parseCoinAmount(tc.input)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("parse %q: got %d want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseCoinAmountRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "-1", "1.000000001", "1.2.3", "abc"} {
		if _, err := parseCoinAmount(input); err == nil {
			t.Fatalf("expected %q to be rejected", input)
		}
	}
}

func TestFormatCoinAmount(t *testing.T) {
	if got := formatCoinAmount(params.CoinDecimals + 1); got != "1.00000001" {
		t.Fatalf("unexpected amount format: %s", got)
	}
}
