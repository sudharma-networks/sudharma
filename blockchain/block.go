package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/transactions"
)

// Block represents a Sudharma Network blockchain block.
type Block struct {
	Version      uint32
	Height       uint64
	Timestamp    int64
	PreviousHash string
	MerkleRoot   string
	Difficulty   uint32
	Nonce        uint64
	MinerAddress string
	Transactions []*transactions.Transaction
}

// HeaderBytes returns the canonical bytes used by Sudharma Network PoW.
//
// MinerAddress is included in the header so a miner cannot
// change the reward destination after mining the block.
func (b *Block) HeaderBytes(nonce uint64) []byte {
	data := make([]byte, 0, 192)

	version := make([]byte, 4)
	binary.BigEndian.PutUint32(version, b.Version)
	data = append(data, version...)

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, b.Height)
	data = append(data, height...)

	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(
		timestamp,
		uint64(b.Timestamp),
	)
	data = append(data, timestamp...)

	data = append(
		data,
		[]byte(b.PreviousHash)...,
	)

	data = append(
		data,
		[]byte(b.MerkleRoot)...,
	)

	// Consensus-critical miner payout destination.
	data = append(
		data,
		[]byte(b.MinerAddress)...,
	)

	difficulty := make([]byte, 4)
	binary.BigEndian.PutUint32(
		difficulty,
		b.Difficulty,
	)
	data = append(data, difficulty...)

	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(
		nonceBytes,
		nonce,
	)
	data = append(data, nonceBytes...)

	return data
}

// Hash calculates the canonical double-SHA256 block hash.
func (b *Block) Hash() string {
	first :=
		sha256.Sum256(
			b.HeaderBytes(b.Nonce),
		)

	second :=
		sha256.Sum256(first[:])

	return hex.EncodeToString(
		second[:],
	)
}

// CalculateMerkleRoot calculates a deterministic Merkle root
// from the transaction IDs in the block.
func (b *Block) CalculateMerkleRoot() string {
	if len(b.Transactions) == 0 {
		hash :=
			sha256.Sum256(
				[]byte(
					"Sudharma Network Empty Transaction Set",
				),
			)

		return hex.EncodeToString(
			hash[:],
		)
	}

	hashes :=
		make(
			[][]byte,
			0,
			len(b.Transactions),
		)

	for _, tx := range b.Transactions {
		if tx == nil {
			continue
		}

		hash, err :=
			hex.DecodeString(tx.ID)

		if err != nil {
			sum :=
				sha256.Sum256(
					[]byte(tx.ID),
				)

			hash = sum[:]
		}

		hashes = append(
			hashes,
			hash,
		)
	}

	if len(hashes) == 0 {
		hash :=
			sha256.Sum256(
				[]byte(
					"Sudharma Network Empty Transaction Set",
				),
			)

		return hex.EncodeToString(
			hash[:],
		)
	}

	for len(hashes) > 1 {
		nextLevel :=
			make(
				[][]byte,
				0,
				(len(hashes)+1)/2,
			)

		for i := 0; i < len(hashes); i += 2 {
			left := hashes[i]
			right := left

			if i+1 < len(hashes) {
				right = hashes[i+1]
			}

			combined :=
				make(
					[]byte,
					0,
					len(left)+len(right),
				)

			combined =
				append(
					combined,
					left...,
				)

			combined =
				append(
					combined,
					right...,
				)

			sum :=
				sha256.Sum256(
					combined,
				)

			nextLevel =
				append(
					nextLevel,
					sum[:],
				)
		}

		hashes = nextLevel
	}

	return hex.EncodeToString(
		hashes[0],
	)
}

// UpdateMerkleRoot recalculates and stores the Merkle root.
func (b *Block) UpdateMerkleRoot() {
	b.MerkleRoot =
		b.CalculateMerkleRoot()
}

// NewGenesisBlock creates the live public-testnet genesis (block 0).
// This identity must not change: altering it would fork the public testnet.
func NewGenesisBlock() *Block {
	return newGenesisBlock(1786924800, "Sudharma Network Genesis Block")
}

// NewMainnetGenesisBlock returns the review-frozen mainnet genesis candidate.
// Nodes must not load this chain until params.MainnetLaunchAuthorized is true.
func NewMainnetGenesisBlock() *Block {
	return newGenesisBlock(
		params.MainnetGenesisTimestamp,
		"Sudharma Network Mainnet Genesis Block v1",
	)
}

// GenesisFor returns the genesis block for a network. Mainnet is refused
// while launch remains unauthorized.
func GenesisFor(network params.NetworkID) (*Block, error) {
	switch network {
	case params.NetworkPublicTestnet:
		return NewGenesisBlock(), nil
	case params.NetworkMainnet:
		if !params.MainnetLaunchAuthorized {
			return nil, fmt.Errorf("mainnet genesis is not authorized")
		}
		return NewMainnetGenesisBlock(), nil
	default:
		return nil, fmt.Errorf("unknown network %q", network)
	}
}

func newGenesisBlock(timestamp uint64, merkleRoot string) *Block {
	block := &Block{
		Version:      1,
		Height:       0,
		Timestamp:    int64(timestamp),
		PreviousHash: "0",
		Difficulty:   1,
		Nonce:        0,
		MinerAddress: "",
		Transactions: nil,
	}
	block.MerkleRoot = merkleRoot
	return block
}

// EncodeUint64 converts an integer to bytes.
func EncodeUint64(value uint64) []byte {
	buffer :=
		make([]byte, 8)

	binary.BigEndian.PutUint64(
		buffer,
		value,
	)

	return buffer
}

// String returns a readable representation of the block.
func (b *Block) String() string {
	return fmt.Sprintf(
		"Block{Height:%d Timestamp:%d Nonce:%d Miner:%s Hash:%s}",
		b.Height,
		b.Timestamp,
		b.Nonce,
		b.MinerAddress,
		b.Hash(),
	)
}
