package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// BlockMessage carries one complete Sudharma Network block
// between connected nodes.
type BlockMessage struct {
	Block *blockchain.Block `json:"block"`
}

// NewBlockMessage creates a P2P message containing
// a complete Sudharma Network block.
func NewBlockMessage(
	block *blockchain.Block,
) ([]byte, error) {

	if block == nil {
		return nil, fmt.Errorf(
			"block cannot be nil",
		)
	}

	return encodeMessage(
		MessageBlock,
		BlockMessage{
			Block: block,
		},
	)
}

// DecodeBlock decodes a block message.
//
// Full consensus validation is intentionally NOT
// performed here. The receiving node must validate
// the block against its own current chain tip and state.
func DecodeBlock(
	message *Message,
) (*blockchain.Block, error) {

	if message == nil {
		return nil, fmt.Errorf(
			"message cannot be nil",
		)
	}

	if message.Type != MessageBlock {
		return nil, fmt.Errorf(
			"message is not a block",
		)
	}

	var payload BlockMessage

	if err := json.Unmarshal(
		message.Payload,
		&payload,
	); err != nil {

		return nil, fmt.Errorf(
			"invalid block message: %w",
			err,
		)
	}

	if payload.Block == nil {
		return nil, fmt.Errorf(
			"block payload is nil",
		)
	}

	return payload.Block, nil
}
