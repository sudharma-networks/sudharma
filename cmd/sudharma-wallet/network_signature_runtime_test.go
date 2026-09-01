package main

import (
	"bytes"
	"os"
	"testing"
)

func TestSendTransactionUsesRPCNetworkForSignature(t *testing.T) {
	source, err := os.ReadFile("commands.go")
	if err != nil {
		t.Fatal(err)
	}

	start := bytes.Index(source, []byte("func sendTransaction()"))
	end := bytes.Index(source, []byte("func showTransactionStatus()"))
	if start < 0 || end <= start {
		t.Fatal("could not isolate sendTransaction implementation")
	}
	body := source[start:end]

	if !bytes.Contains(body, []byte("client.Status(ctx)")) {
		t.Fatal("sendTransaction must resolve the RPC node network before signing")
	}
	if !bytes.Contains(body, []byte("params.ParseNetwork(status.Network)")) {
		t.Fatal("sendTransaction must validate the reported RPC network through params.ParseNetwork")
	}
	if bytes.Contains(body, []byte("tx.Sign(w)")) {
		t.Fatal("sendTransaction must not use the default-network signer")
	}
	if !bytes.Contains(body, []byte("tx.SignForNetwork(w, network)")) {
		t.Fatal("sendTransaction must sign with the validated active RPC network")
	}
}
