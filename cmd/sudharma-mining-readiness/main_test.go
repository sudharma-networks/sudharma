package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/sudharma-networks/sudharma/params"
)

func TestMiningReadinessJSONReportsTestnetStack(t *testing.T) {
	payload := buildOutput()
	if !payload.StackReady {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.MainnetMiningAuthorized {
		t.Fatal("mainnet mining must stay unauthorized")
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		t.Fatal(err)
	}
	if !params.MiningStackReady() {
		t.Fatal("MiningStackReady must be true")
	}
}
