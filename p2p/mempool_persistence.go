package p2p

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sudharma-networks/sudharma/blockchain"
	"github.com/sudharma-networks/sudharma/transactions"
)

const (
	MempoolFileVersion uint32 = 1
)

type persistedMempool struct {
	Version      uint32                      `json:"version"`
	Transactions []*transactions.Transaction `json:"transactions"`
}

// SaveMempoolToFile saves all currently pending transactions.
//
// The file is written atomically:
//  1. write temporary file
//  2. fsync/close
//  3. replace final file
func (n *Node) SaveMempoolToFile(
	path string,
) error {

	if n == nil {
		return fmt.Errorf(
			"node cannot be nil",
		)
	}

	if path == "" {
		return fmt.Errorf(
			"mempool path cannot be empty",
		)
	}

	txs :=
		n.mempool.AllTransactions()

	// Stable ordering makes the saved file deterministic.
	sort.Slice(
		txs,
		func(i, j int) bool {

			if txs[i].From != txs[j].From {
				return txs[i].From <
					txs[j].From
			}

			if txs[i].Nonce != txs[j].Nonce {
				return txs[i].Nonce <
					txs[j].Nonce
			}

			return txs[i].ID <
				txs[j].ID
		},
	)

	payload :=
		persistedMempool{
			Version:      MempoolFileVersion,
			Transactions: txs,
		}

	data, err :=
		json.MarshalIndent(
			payload,
			"",
			"  ",
		)

	if err != nil {
		return fmt.Errorf(
			"failed encoding mempool: %w",
			err,
		)
	}

	directory :=
		filepath.Dir(path)

	if err :=
		os.MkdirAll(
			directory,
			0o755,
		); err != nil {

		return fmt.Errorf(
			"failed creating mempool directory: %w",
			err,
		)
	}

	tempPath :=
		path + ".tmp"

	file, err :=
		os.OpenFile(
			tempPath,
			os.O_CREATE|
				os.O_WRONLY|
				os.O_TRUNC,
			0o600,
		)

	if err != nil {
		return fmt.Errorf(
			"failed creating temporary mempool file: %w",
			err,
		)
	}

	writeFailed := false

	if _, err :=
		file.Write(data); err != nil {

		writeFailed = true
		_ = file.Close()
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed writing mempool file: %w",
			err,
		)
	}

	if err :=
		file.Sync(); err != nil {

		writeFailed = true
		_ = file.Close()
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed syncing mempool file: %w",
			err,
		)
	}

	if err :=
		file.Close(); err != nil {

		writeFailed = true
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed closing mempool file: %w",
			err,
		)
	}

	if writeFailed {
		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed writing mempool file",
		)
	}

	// Windows cannot always rename over an existing file,
	// so remove the old destination first.
	if err :=
		os.Remove(path); err != nil &&
		!os.IsNotExist(err) {

		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed replacing old mempool file: %w",
			err,
		)
	}

	if err :=
		os.Rename(
			tempPath,
			path,
		); err != nil {

		_ = os.Remove(tempPath)

		return fmt.Errorf(
			"failed installing mempool file: %w",
			err,
		)
	}

	return nil
}

// LoadMempoolFromFile loads pending transactions from disk.
//
// Every stored transaction is revalidated against:
//   - the current confirmed blockchain state
//   - transactions already restored from the same file
//
// This is important because a transaction that was valid when the
// node shut down may no longer be valid after a chain reorganization.
//
// loaded   = transactions restored successfully
// rejected = stored transactions discarded as invalid
func (n *Node) LoadMempoolFromFile(
	path string,
) (
	loaded int,
	rejected int,
	err error,
) {

	if n == nil {
		return 0, 0, fmt.Errorf(
			"node cannot be nil",
		)
	}

	if path == "" {
		return 0, 0, fmt.Errorf(
			"mempool path cannot be empty",
		)
	}

	state :=
		n.State()

	if state == nil {
		return 0, 0, fmt.Errorf(
			"blockchain state is not attached",
		)
	}

	data, err :=
		os.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}

		return 0, 0, fmt.Errorf(
			"failed reading mempool file: %w",
			err,
		)
	}

	var payload persistedMempool

	if err :=
		json.Unmarshal(
			data,
			&payload,
		); err != nil {

		return 0, 0, fmt.Errorf(
			"invalid mempool file: %w",
			err,
		)
	}

	if payload.Version !=
		MempoolFileVersion {

		return 0, 0, fmt.Errorf(
			"unsupported mempool file version: %d",
			payload.Version,
		)
	}

	ordered :=
		make(
			[]*transactions.Transaction,
			0,
			len(payload.Transactions),
		)

	seen :=
		make(
			map[string]bool,
		)

	for _, tx := range payload.Transactions {

		if tx == nil ||
			tx.ID == "" {

			rejected++
			continue
		}

		if seen[tx.ID] {
			rejected++
			continue
		}

		seen[tx.ID] = true

		ordered =
			append(
				ordered,
				tx,
			)
	}

	sort.Slice(
		ordered,
		func(i, j int) bool {

			if ordered[i].From !=
				ordered[j].From {

				return ordered[i].From <
					ordered[j].From
			}

			if ordered[i].Nonce !=
				ordered[j].Nonce {

				return ordered[i].Nonce <
					ordered[j].Nonce
			}

			return ordered[i].ID <
				ordered[j].ID
		},
	)

	// Build a validated replacement set first.
	//
	// We do not modify the live mempool until all file decoding
	// and structural checks have completed.
	valid :=
		make(
			[]*transactions.Transaction,
			0,
			len(ordered),
		)

	for _, tx := range ordered {

		if state.IsTransactionProcessed(
			tx.ID,
		) {
			rejected++
			continue
		}

		if !tx.Verify() {
			rejected++
			continue
		}

		if err :=
			blockchain.ValidateMempoolTransaction(
				state,
				valid,
				tx,
			); err != nil {

			rejected++

			fmt.Printf(
				"[MEMPOOL] Stored transaction %s rejected: %v\n",
				tx.ID,
				err,
			)

			continue
		}

		valid =
			append(
				valid,
				tx,
			)
	}

	n.mempool.Clear()

	for _, tx := range valid {

		if err :=
			n.mempool.AddTransaction(
				tx,
			); err != nil {

			return loaded,
				rejected,
				fmt.Errorf(
					"failed restoring transaction %s: %w",
					tx.ID,
					err,
				)
		}

		loaded++
	}

	return loaded,
		rejected,
		nil
}
