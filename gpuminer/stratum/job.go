package stratum

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// Job is parsed from a Sudharma Stratum mining.notify message.
type Job struct {
	ID              string
	Height          uint64
	Parent          string
	MerkleRoot      string
	PoolDifficulty  uint32
	BlockDifficulty uint32
	PoolTarget      string
	Timestamp       int64
	Version         uint32
	MinerAddress    string
	Clean           bool
	Block           blockchain.Block
}

// ParseNotify converts Sudharma pool mining.notify params into a mineable job.
//
// Params: job_id, height, parent, merkle_root, pool_difficulty, block_difficulty,
// pool_target, timestamp, version, miner_address, clean_jobs
func ParseNotify(params []any) (Job, error) {
	if len(params) < 11 {
		return Job{}, fmt.Errorf("mining.notify requires 11 params, got %d", len(params))
	}
	id, err := paramString(params, 0)
	if err != nil {
		return Job{}, err
	}
	height, err := paramUint64(params, 1)
	if err != nil {
		return Job{}, err
	}
	parent, err := paramString(params, 2)
	if err != nil {
		return Job{}, err
	}
	merkle, err := paramString(params, 3)
	if err != nil {
		return Job{}, err
	}
	poolDiff, err := paramUint32(params, 4)
	if err != nil {
		return Job{}, err
	}
	blockDiff, err := paramUint32(params, 5)
	if err != nil {
		return Job{}, err
	}
	target, err := paramString(params, 6)
	if err != nil {
		return Job{}, err
	}
	timestamp, err := paramInt64(params, 7)
	if err != nil {
		return Job{}, err
	}
	version, err := paramUint32(params, 8)
	if err != nil {
		return Job{}, err
	}
	minerAddress, err := paramString(params, 9)
	if err != nil {
		return Job{}, err
	}
	clean, err := paramBool(params, 10)
	if err != nil {
		return Job{}, err
	}

	block := blockchain.Block{
		Version:      version,
		Height:       height,
		Timestamp:    timestamp,
		PreviousHash: parent,
		MerkleRoot:   merkle,
		Difficulty:   blockDiff,
		MinerAddress: strings.ToLower(strings.TrimSpace(minerAddress)),
	}

	return Job{
		ID:              id,
		Height:          height,
		Parent:          parent,
		MerkleRoot:      merkle,
		PoolDifficulty:  poolDiff,
		BlockDifficulty: blockDiff,
		PoolTarget:      target,
		Timestamp:       timestamp,
		Version:         version,
		MinerAddress:    block.MinerAddress,
		Clean:           clean,
		Block:           block,
	}, nil
}

func paramString(params []any, index int) (string, error) {
	if index >= len(params) {
		return "", fmt.Errorf("missing param %d", index)
	}
	switch v := params[index].(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatInt(int64(v), 10), nil
	default:
		return fmt.Sprint(v), nil
	}
}

func paramUint64(params []any, index int) (uint64, error) {
	raw, err := paramString(params, index)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid uint64 param %d: %q", index, raw)
	}
	return value, nil
}

func paramUint32(params []any, index int) (uint32, error) {
	value, err := paramUint64(params, index)
	if err != nil {
		return 0, err
	}
	if value > uint64(^uint32(0)) {
		return 0, fmt.Errorf("param %d overflows uint32", index)
	}
	return uint32(value), nil
}

func paramInt64(params []any, index int) (int64, error) {
	raw, err := paramString(params, index)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64 param %d: %q", index, raw)
	}
	return value, nil
}

func paramBool(params []any, index int) (bool, error) {
	raw, err := paramString(params, index)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(raw) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool param %d: %q", index, raw)
	}
}
