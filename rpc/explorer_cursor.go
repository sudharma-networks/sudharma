package rpc

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sudharma-networks/sudharma/blockchain"
)

const (
	maxExplorerCursorBytes = 256
	maxExplorerScanBlocks  = 2_000
)

var (
	errExplorerInvalidCursor = errors.New("invalid explorer cursor")
	errExplorerStaleCursor   = errors.New("explorer cursor is stale")
)

type explorerCursor struct {
	Height    uint64
	Offset    uint64
	BlockHash string
}

func encodeExplorerCursor(cursor explorerCursor) string {
	payload := fmt.Sprintf("%d:%d:%s", cursor.Height, cursor.Offset, cursor.BlockHash)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeExplorerCursor(raw string) (explorerCursor, error) {
	if raw == "" || len(raw) > maxExplorerCursorBytes {
		return explorerCursor{}, errExplorerInvalidCursor
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(payload) == 0 || len(payload) > maxExplorerCursorBytes {
		return explorerCursor{}, errExplorerInvalidCursor
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 || len(parts[2]) != 64 || !isLowerHex(parts[2]) {
		return explorerCursor{}, errExplorerInvalidCursor
	}
	height, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return explorerCursor{}, errExplorerInvalidCursor
	}
	offset, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return explorerCursor{}, errExplorerInvalidCursor
	}
	return explorerCursor{Height: height, Offset: offset, BlockHash: parts[2]}, nil
}

// explorerConfirmedPage returns a bounded page of canonical confirmed
// transactions. The opaque cursor includes the canonical block hash so a
// reorganization invalidates old pagination state instead of silently mixing
// two different histories.
func (s *Server) explorerConfirmedPage(address string, limit int, rawCursor string, beforeHeight *uint64) ([]blockchain.ConfirmedTransaction, string, error) {
	if s == nil || s.chain == nil || limit <= 0 {
		return nil, "", errExplorerInvalidCursor
	}
	if rawCursor != "" && beforeHeight != nil {
		return nil, "", errExplorerInvalidCursor
	}

	chainHeight := s.chain.Height()
	startHeight := chainHeight
	startOffset := uint64(0)

	if rawCursor != "" {
		cursor, err := decodeExplorerCursor(rawCursor)
		if err != nil || cursor.Height > chainHeight {
			return nil, "", errExplorerInvalidCursor
		}
		block, ok := s.chain.BlockByHeight(cursor.Height)
		if !ok || block == nil {
			return nil, "", errExplorerStaleCursor
		}
		if block.Hash() != cursor.BlockHash {
			return nil, "", errExplorerStaleCursor
		}
		if cursor.Offset > uint64(len(block.Transactions)) {
			return nil, "", errExplorerInvalidCursor
		}
		startHeight = cursor.Height
		startOffset = cursor.Offset
	} else if beforeHeight != nil {
		if *beforeHeight == 0 {
			return []blockchain.ConfirmedTransaction{}, "", nil
		}
		candidate := *beforeHeight - 1
		if candidate < startHeight {
			startHeight = candidate
		}
	}

	result := make([]blockchain.ConfirmedTransaction, 0, limit)
	scannedBlocks := 0

	for height := startHeight; ; height-- {
		block, ok := s.chain.BlockByHeight(height)
		if !ok || block == nil {
			return nil, "", errExplorerStaleCursor
		}
		blockHash := block.Hash()
		offset := uint64(0)
		if height == startHeight {
			offset = startOffset
		}

		for index := offset; index < uint64(len(block.Transactions)); index++ {
			tx := block.Transactions[index]
			if tx == nil || (address != "" && tx.From != address && tx.To != address) {
				continue
			}
			if len(result) == limit {
				return result, encodeExplorerCursor(explorerCursor{
					Height:    height,
					Offset:    index,
					BlockHash: blockHash,
				}), nil
			}
			result = append(result, blockchain.ConfirmedTransaction{
				Transaction:    tx,
				BlockHeight:    height,
				BlockHash:      blockHash,
				BlockTimestamp: block.Timestamp,
			})
		}

		scannedBlocks++
		if height == 0 {
			break
		}
		if scannedBlocks >= maxExplorerScanBlocks {
			nextHeight := height - 1
			nextBlock, ok := s.chain.BlockByHeight(nextHeight)
			if !ok || nextBlock == nil {
				return nil, "", errExplorerStaleCursor
			}
			return result, encodeExplorerCursor(explorerCursor{
				Height:    nextHeight,
				Offset:    0,
				BlockHash: nextBlock.Hash(),
			}), nil
		}
	}

	return result, "", nil
}
