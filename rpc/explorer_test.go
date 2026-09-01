package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/miner"
	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
	"github.com/sudharma-networks/sudharma/wallet"
)

type explorerBlockJSON struct {
	Height           uint64 `json:"height"`
	Hash             string `json:"hash"`
	Timestamp        int64  `json:"timestamp"`
	PreviousHash     string `json:"previous_hash"`
	MerkleRoot       string `json:"merkle_root"`
	Difficulty       uint32 `json:"difficulty"`
	Nonce            uint64 `json:"nonce"`
	MinerAddress     string `json:"miner_address"`
	TransactionCount int    `json:"transaction_count"`
	Transactions     []struct {
		ID     string `json:"id"`
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
		Fee    uint64 `json:"fee"`
		Nonce  uint64 `json:"nonce"`
	} `json:"transactions,omitempty"`
}

type explorerTransactionJSON struct {
	Transaction struct {
		ID     string `json:"id"`
		From   string `json:"from"`
		To     string `json:"to"`
		Amount uint64 `json:"amount"`
		Fee    uint64 `json:"fee"`
		Nonce  uint64 `json:"nonce"`
	} `json:"transaction"`
	Status         string  `json:"status"`
	BlockHeight    *uint64 `json:"block_height,omitempty"`
	BlockHash      string  `json:"block_hash,omitempty"`
	BlockTimestamp int64   `json:"block_timestamp,omitempty"`
	Confirmations  uint64  `json:"confirmations"`
}

func addExplorerRPCBlock(t *testing.T, chain *blockchain.Chain, txs ...*transactions.Transaction) *blockchain.Block {
	t.Helper()
	previous := chain.Tip()
	block := &blockchain.Block{
		Version:      1,
		Height:       previous.Height + 1,
		Timestamp:    previous.Timestamp + int64(params.TargetBlockTimeSeconds),
		PreviousHash: previous.Hash(),
		Difficulty:   previous.Difficulty,
		Transactions: txs,
	}
	block.UpdateMerkleRoot()
	result := miner.Mine(block, 0, 10)
	if !result.Found {
		t.Fatal("failed to mine explorer RPC fixture block")
	}
	if err := chain.AddBlock(block); err != nil {
		t.Fatalf("failed to add explorer RPC fixture block: %v", err)
	}
	return block
}

func TestExplorerStatusUsesCurrentCanonicalChain(t *testing.T) {
	server, node, chain, state := newTestServer(t)
	if err := state.MintSupply(123); err != nil {
		t.Fatal(err)
	}
	addExplorerRPCBlock(t, chain)

	response := request(t, server, http.MethodGet, "/v1/explorer/status", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("explorer status returned %d: %s", response.Code, response.Body.String())
	}
	var got struct {
		Network      string `json:"network"`
		Coin         string `json:"coin"`
		Symbol       string `json:"symbol"`
		Height       uint64 `json:"height"`
		TipHash      string `json:"tip_hash"`
		TotalWork    string `json:"total_work"`
		Peers        int    `json:"peers"`
		Mempool      int    `json:"mempool"`
		IssuedSupply uint64 `json:"issued_supply"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Network != "sudharma" || got.Symbol != params.CoinSymbol || got.Height != chain.Height() || got.TipHash != chain.Tip().Hash() {
		t.Fatalf("unexpected explorer status: %+v", got)
	}
	if got.Peers != node.PeerCount() || got.Mempool != node.MempoolCount() || got.IssuedSupply != 123 || got.TotalWork == "" {
		t.Fatalf("unexpected explorer counters: %+v", got)
	}
}

func TestExplorerRecentBlocksAndLookupByHeightOrHash(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	block1 := addExplorerRPCBlock(t, chain)
	block2 := addExplorerRPCBlock(t, chain)

	response := request(t, server, http.MethodGet, "/v1/explorer/blocks?limit=2", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("recent blocks returned %d: %s", response.Code, response.Body.String())
	}
	var list struct {
		Blocks     []explorerBlockJSON `json:"blocks"`
		NextBefore *uint64             `json:"next_before,omitempty"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Blocks) != 2 || list.Blocks[0].Height != block2.Height || list.Blocks[1].Height != block1.Height {
		t.Fatalf("unexpected recent blocks: %+v", list.Blocks)
	}
	if list.Blocks[0].Hash != block2.Hash() || list.NextBefore == nil || *list.NextBefore != block1.Height {
		t.Fatalf("unexpected block hash/cursor: %+v", list)
	}

	for _, id := range []string{fmt.Sprint(block1.Height), block1.Hash()} {
		detail := request(t, server, http.MethodGet, "/v1/explorer/blocks/"+id, nil, "")
		if detail.Code != http.StatusOK {
			t.Fatalf("block lookup %q returned %d: %s", id, detail.Code, detail.Body.String())
		}
		var got explorerBlockJSON
		if err := json.Unmarshal(detail.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Height != block1.Height || got.Hash != block1.Hash() {
			t.Fatalf("unexpected block detail for %q: %+v", id, got)
		}
	}
}

func TestExplorerConfirmedAndPendingTransactionStatus(t *testing.T) {
	server, node, chain, state := newTestServer(t)
	confirmed := transactions.NewTransaction("1111111111111111111111111111111111111111", "2222222222222222222222222222222222222222", params.MinTransferAmount, 1)
	block := addExplorerRPCBlock(t, chain, confirmed)

	confirmedResponse := request(t, server, http.MethodGet, "/v1/explorer/transactions/"+confirmed.ID, nil, "")
	if confirmedResponse.Code != http.StatusOK {
		t.Fatalf("confirmed explorer tx returned %d: %s", confirmedResponse.Code, confirmedResponse.Body.String())
	}
	var confirmedGot explorerTransactionJSON
	if err := json.Unmarshal(confirmedResponse.Body.Bytes(), &confirmedGot); err != nil {
		t.Fatal(err)
	}
	if confirmedGot.Status != "confirmed" || confirmedGot.BlockHeight == nil || *confirmedGot.BlockHeight != block.Height || confirmedGot.BlockHash != block.Hash() || confirmedGot.Confirmations != 1 {
		t.Fatalf("unexpected confirmed explorer tx: %+v", confirmedGot)
	}

	sender, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := state.Credit(sender.Address, 10*params.CoinDecimals); err != nil {
		t.Fatal(err)
	}
	pending := transactions.NewTransaction(sender.Address, receiver.Address, params.CoinDecimals, 1)
	if err := pending.Sign(sender); err != nil {
		t.Fatal(err)
	}
	if _, err := node.SubmitTransaction(pending); err != nil {
		t.Fatalf("failed to submit pending fixture: %v", err)
	}
	pendingResponse := request(t, server, http.MethodGet, "/v1/explorer/transactions/"+pending.ID, nil, "")
	if pendingResponse.Code != http.StatusOK {
		t.Fatalf("pending explorer tx returned %d: %s", pendingResponse.Code, pendingResponse.Body.String())
	}
	var pendingGot explorerTransactionJSON
	if err := json.Unmarshal(pendingResponse.Body.Bytes(), &pendingGot); err != nil {
		t.Fatal(err)
	}
	if pendingGot.Status != "pending" || pendingGot.BlockHeight != nil || pendingGot.Confirmations != 0 {
		t.Fatalf("unexpected pending explorer tx: %+v", pendingGot)
	}
}

func TestExplorerAddressDetailsAndHistory(t *testing.T) {
	server, _, chain, state := newTestServer(t)
	address := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := state.Credit(address, 98765); err != nil {
		t.Fatal(err)
	}
	if err := state.SetAccountNonce(address, 3); err != nil {
		t.Fatal(err)
	}
	incoming := transactions.NewTransaction("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", address, params.MinTransferAmount, 1)
	outgoing := transactions.NewTransaction(address, "cccccccccccccccccccccccccccccccccccccccc", params.MinTransferAmount, 4)
	addExplorerRPCBlock(t, chain, incoming, outgoing)

	response := request(t, server, http.MethodGet, "/v1/explorer/addresses/"+address+"?limit=10", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("address explorer returned %d: %s", response.Code, response.Body.String())
	}
	var got struct {
		Address        string                    `json:"address"`
		Balance        uint64                    `json:"balance"`
		ConfirmedNonce uint64                    `json:"confirmed_nonce"`
		NextNonce      uint64                    `json:"next_nonce"`
		Transactions   []explorerTransactionJSON `json:"transactions"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Address != address || got.Balance != 98765 || got.ConfirmedNonce != 3 || got.NextNonce != 4 || len(got.Transactions) != 2 {
		t.Fatalf("unexpected address explorer response: %+v", got)
	}
}

func TestExplorerSearchResolvesBlockTransactionAndAddress(t *testing.T) {
	server, _, chain, _ := newTestServer(t)
	address := "dddddddddddddddddddddddddddddddddddddddd"
	tx := transactions.NewTransaction(address, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", params.MinTransferAmount, 1)
	block := addExplorerRPCBlock(t, chain, tx)

	cases := []struct {
		query string
		kind  string
		path  string
	}{
		{query: fmt.Sprint(block.Height), kind: "block", path: "/explorer/block?id=1"},
		{query: tx.ID, kind: "transaction", path: "/explorer/tx?id=" + tx.ID},
		{query: block.Hash(), kind: "block", path: "/explorer/block?id=" + block.Hash()},
		{query: address, kind: "address", path: "/explorer/address?address=" + address},
	}
	for _, tc := range cases {
		response := request(t, server, http.MethodGet, "/v1/explorer/search?q="+tc.query, nil, "")
		if response.Code != http.StatusOK {
			t.Fatalf("search %q returned %d: %s", tc.query, response.Code, response.Body.String())
		}
		var got struct {
			Type string `json:"type"`
			Path string `json:"path"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Type != tc.kind || got.Path != tc.path {
			t.Fatalf("search %q = %+v, want type=%q path=%q", tc.query, got, tc.kind, tc.path)
		}
	}

	missing := request(t, server, http.MethodGet, "/v1/explorer/search?q=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", nil, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing search returned %d", missing.Code)
	}
}

func TestExplorerReadOnlyAndPaginationHardening(t *testing.T) {
	server, _, _, _ := newTestServer(t)

	post := request(t, server, http.MethodPost, "/v1/explorer/status", nil, "")
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("explorer POST handling = %d allow=%q", post.Code, post.Header().Get("Allow"))
	}

	badLimit := request(t, server, http.MethodGet, "/v1/explorer/blocks?limit=101", nil, "")
	if badLimit.Code != http.StatusBadRequest {
		t.Fatalf("explorer limit=101 returned %d", badLimit.Code)
	}

	badBlock := request(t, server, http.MethodGet, "/v1/explorer/blocks/not-a-height-or-hash", nil, "")
	if badBlock.Code != http.StatusBadRequest {
		t.Fatalf("invalid explorer block ID returned %d", badBlock.Code)
	}

	emptySearch := request(t, server, http.MethodGet, "/v1/explorer/search?q=", nil, "")
	if emptySearch.Code != http.StatusBadRequest {
		t.Fatalf("empty explorer search returned %d", emptySearch.Code)
	}
}
