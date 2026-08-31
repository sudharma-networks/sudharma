package rpc

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/sudharma-networks/sudharma/transactions"
)

func TestExplorerTransactionCursorDoesNotSkipRemainderOfBlock(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	tx1 := transactions.NewTransaction("1111111111111111111111111111111111111111", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1, 1)
	tx2 := transactions.NewTransaction("2222222222222222222222222222222222222222", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 2, 1)
	tx3 := transactions.NewTransaction("3333333333333333333333333333333333333333", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 3, 1)
	addExplorerRPCBlock(t, chain, tx1, tx2, tx3)

	first := request(t, server, http.MethodGet, "/v1/explorer/transactions?limit=2", nil, "")
	if first.Code != http.StatusOK {
		t.Fatalf("first transaction page returned %d: %s", first.Code, first.Body.String())
	}
	var page1 struct {
		Transactions []explorerTransactionJSON `json:"transactions"`
		NextCursor   string                    `json:"next_cursor"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &page1); err != nil {
		t.Fatal(err)
	}
	if len(page1.Transactions) != 2 || page1.NextCursor == "" {
		t.Fatalf("first page = %+v, want 2 transactions and a cursor", page1)
	}

	secondPath := "/v1/explorer/transactions?limit=2&cursor=" + url.QueryEscape(page1.NextCursor)
	second := request(t, server, http.MethodGet, secondPath, nil, "")
	if second.Code != http.StatusOK {
		t.Fatalf("second transaction page returned %d: %s", second.Code, second.Body.String())
	}
	var page2 struct {
		Transactions []explorerTransactionJSON `json:"transactions"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &page2); err != nil {
		t.Fatal(err)
	}
	if len(page2.Transactions) != 1 || page2.Transactions[0].Transaction.ID != tx3.ID {
		t.Fatalf("second page = %+v, want only third transaction %s", page2, tx3.ID)
	}
}

func TestExplorerAddressCursorDoesNotSkipRemainderOfBlock(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tx1 := transactions.NewTransaction(address, "1111111111111111111111111111111111111111", 1, 1)
	tx2 := transactions.NewTransaction("2222222222222222222222222222222222222222", address, 2, 1)
	tx3 := transactions.NewTransaction(address, "3333333333333333333333333333333333333333", 3, 2)
	addExplorerRPCBlock(t, chain, tx1, tx2, tx3)

	firstPath := "/v1/explorer/addresses/" + address + "?limit=2"
	first := request(t, server, http.MethodGet, firstPath, nil, "")
	if first.Code != http.StatusOK {
		t.Fatalf("first address page returned %d: %s", first.Code, first.Body.String())
	}
	var page1 struct {
		Transactions []explorerTransactionJSON `json:"transactions"`
		NextCursor   string                    `json:"next_cursor"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &page1); err != nil {
		t.Fatal(err)
	}
	if len(page1.Transactions) != 2 || page1.NextCursor == "" {
		t.Fatalf("first address page = %+v, want 2 transactions and a cursor", page1)
	}

	secondPath := firstPath + "&cursor=" + url.QueryEscape(page1.NextCursor)
	second := request(t, server, http.MethodGet, secondPath, nil, "")
	if second.Code != http.StatusOK {
		t.Fatalf("second address page returned %d: %s", second.Code, second.Body.String())
	}
	var page2 struct {
		Transactions []explorerTransactionJSON `json:"transactions"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &page2); err != nil {
		t.Fatal(err)
	}
	if len(page2.Transactions) != 1 || page2.Transactions[0].Transaction.ID != tx3.ID {
		t.Fatalf("second address page = %+v, want only third transaction %s", page2, tx3.ID)
	}
}
