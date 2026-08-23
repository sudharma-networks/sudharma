package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
)

const MaxBlocksPerMessage uint64 = 128

// GetBlocksMessage asks a peer for blocks beginning
// at StartHeight.
type GetBlocksMessage struct {
	StartHeight uint64 `json:"start_height"`
	Limit       uint64 `json:"limit"`
}

// BlocksMessage carries a consecutive batch of blocks.
type BlocksMessage struct {
	Blocks []*blockchain.Block `json:"blocks"`
}

// NewGetBlocksMessage creates a request for a range
// of blocks starting at startHeight.
func NewGetBlocksMessage(
	startHeight uint64,
	limit uint64,
) ([]byte, error) {

	if limit == 0 {
		return nil, fmt.Errorf(
			"block request limit cannot be zero",
		)
	}

	if limit > MaxBlocksPerMessage {
		return nil, fmt.Errorf(
			"block request limit exceeds maximum %d",
			MaxBlocksPerMessage,
		)
	}

	return encodeMessage(
		MessageGetBlocks,
		GetBlocksMessage{
			StartHeight: startHeight,
			Limit:       limit,
		},
	)
}

// DecodeGetBlocks decodes a block-range request.
func DecodeGetBlocks(
	message *Message,
) (*GetBlocksMessage, error) {

	if message == nil {
		return nil, fmt.Errorf(
			"message cannot be nil",
		)
	}

	if message.Type != MessageGetBlocks {
		return nil, fmt.Errorf(
			"message is not a get_blocks request",
		)
	}

	var request GetBlocksMessage

	if err := json.Unmarshal(
		message.Payload,
		&request,
	); err != nil {

		return nil, fmt.Errorf(
			"invalid get_blocks message: %w",
			err,
		)
	}

	if request.Limit == 0 {
		return nil, fmt.Errorf(
			"block request limit cannot be zero",
		)
	}

	if request.Limit > MaxBlocksPerMessage {
		return nil, fmt.Errorf(
			"block request limit exceeds maximum %d",
			MaxBlocksPerMessage,
		)
	}

	return &request, nil
}

// NewBlocksMessage creates a response containing
// consecutive Sudharma Network blocks.
func NewBlocksMessage(
	blocks []*blockchain.Block,
) ([]byte, error) {

	if len(blocks) == 0 {
		return nil, fmt.Errorf(
			"blocks response cannot be empty",
		)
	}

	if uint64(len(blocks)) >
		MaxBlocksPerMessage {

		return nil, fmt.Errorf(
			"blocks response exceeds maximum %d",
			MaxBlocksPerMessage,
		)
	}

	for i, block := range blocks {
		if block == nil {
			return nil, fmt.Errorf(
				"block %d is nil",
				i,
			)
		}
	}

	return encodeMessage(
		MessageBlocks,
		BlocksMessage{
			Blocks: blocks,
		},
	)
}

// DecodeBlocks decodes a batch of blocks.
//
// Consensus validation is not performed here.
// The receiving node must validate every block
// against its own chain and state.
func DecodeBlocks(
	message *Message,
) ([]*blockchain.Block, error) {

	if message == nil {
		return nil, fmt.Errorf(
			"message cannot be nil",
		)
	}

	if message.Type != MessageBlocks {
		return nil, fmt.Errorf(
			"message is not a blocks response",
		)
	}

	var response BlocksMessage

	if err := json.Unmarshal(
		message.Payload,
		&response,
	); err != nil {

		return nil, fmt.Errorf(
			"invalid blocks message: %w",
			err,
		)
	}

	if len(response.Blocks) == 0 {
		return nil, fmt.Errorf(
			"blocks response cannot be empty",
		)
	}

	if uint64(len(response.Blocks)) >
		MaxBlocksPerMessage {

		return nil, fmt.Errorf(
			"blocks response exceeds maximum %d",
			MaxBlocksPerMessage,
		)
	}

	for i, block := range response.Blocks {
		if block == nil {
			return nil, fmt.Errorf(
				"block %d is nil",
				i,
			)
		}
	}

	for i := 1; i < len(response.Blocks); i++ {
		previous :=
			response.Blocks[i-1]

		current :=
			response.Blocks[i]

		if current.Height !=
			previous.Height+1 {

			return nil, fmt.Errorf(
				"non-consecutive blocks: expected height %d, got %d",
				previous.Height+1,
				current.Height,
			)
		}

		if current.PreviousHash !=
			previous.Hash() {

			return nil, fmt.Errorf(
				"block %d does not link to previous block",
				current.Height,
			)
		}
	}

	return response.Blocks, nil
}
