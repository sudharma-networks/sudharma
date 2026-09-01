package main

import (
	"bytes"
	"os"
	"testing"
)

func TestDevelopmentTransactionHelpersUseExplicitNetworkSigner(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(source, []byte("tx.Sign(")) {
		t.Fatal("sudharmad transaction helpers must not use the default-network signer")
	}
	if got := bytes.Count(source, []byte("tx.SignForNetwork(")); got < 3 {
		t.Fatalf("expected every sudharmad transaction helper to use SignForNetwork, got %d explicit signer call(s)", got)
	}
}
