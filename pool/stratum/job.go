package stratum

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
)

const maxStaleJobIDs = 8

const (
	jobDomain  = "SUDHARMA-STRATUM-JOB-V1\x00"
	laneDomain = "SUDHARMA-STRATUM-LANE-V1\x00"
)

type job struct {
	id         string
	work       Work
	generation uint64
	lane       uint32
}

type notifyParams struct {
	JobID             string
	Algorithm         string
	Height            uint64
	TargetHex         string
	HeaderPrefixHex   string
	RewardAddress     string
	Version           uint32
	NetworkDifficulty uint32
	Lane              uint32
	CleanJobs         bool
}

func (p notifyParams) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		p.JobID,
		p.Algorithm,
		p.Height,
		p.TargetHex,
		p.HeaderPrefixHex,
		p.RewardAddress,
		p.Version,
		p.NetworkDifficulty,
		p.Lane,
		p.CleanJobs,
	})
}

type laneSessionKey struct {
	workID    string
	sessionID string
}

type LaneAllocator struct {
	mu        sync.Mutex
	byWork    map[string]map[uint32]string
	bySession map[laneSessionKey]uint32
}

func NewLaneAllocator() *LaneAllocator {
	return &LaneAllocator{
		byWork:    make(map[string]map[uint32]string),
		bySession: make(map[laneSessionKey]uint32),
	}
}

func (a *LaneAllocator) Acquire(workID, sessionID string) (uint32, error) {
	if a == nil {
		return 0, errors.New("nil lane allocator")
	}
	if workID == "" || sessionID == "" {
		return 0, errors.New("work ID and session ID are required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	key := laneSessionKey{workID: workID, sessionID: sessionID}
	if lane, ok := a.bySession[key]; ok {
		return lane, nil
	}
	lanes := a.byWork[workID]
	if lanes == nil {
		lanes = make(map[uint32]string)
		a.byWork[workID] = lanes
	}

	start := laneStart(workID, sessionID)
	for offset := uint64(0); offset <= uint64(math.MaxUint32); offset++ {
		lane := start + uint32(offset)
		if _, occupied := lanes[lane]; occupied {
			continue
		}
		lanes[lane] = sessionID
		a.bySession[key] = lane
		return lane, nil
	}
	return 0, errors.New("nonce lane space exhausted")
}

func (a *LaneAllocator) Release(workID, sessionID string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	key := laneSessionKey{workID: workID, sessionID: sessionID}
	lane, ok := a.bySession[key]
	if !ok {
		return
	}
	delete(a.bySession, key)
	if lanes := a.byWork[workID]; lanes != nil {
		if lanes[lane] == sessionID {
			delete(lanes, lane)
		}
		if len(lanes) == 0 {
			delete(a.byWork, workID)
		}
	}
}

func laneStart(workID, sessionID string) uint32 {
	h := sha256.New()
	_, _ = h.Write([]byte(laneDomain))
	_, _ = h.Write([]byte(workID))
	_, _ = h.Write([]byte(sessionID))
	sum := h.Sum(nil)
	return binary.BigEndian.Uint32(sum[:4])
}

func deriveJobID(workID, sessionID string, generation uint64) string {
	h := sha256.New()
	_, _ = h.Write([]byte(jobDomain))
	_, _ = h.Write([]byte(workID))
	_, _ = h.Write([]byte(sessionID))
	var generationBytes [8]byte
	binary.BigEndian.PutUint64(generationBytes[:], generation)
	_, _ = h.Write(generationBytes[:])
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Session) RefreshWork(ctx context.Context) ([]Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	if s.identity == nil {
		s.mu.Unlock()
		return nil, newProtocolError(protocolInvalidRequest)
	}
	rewardAddress := s.identity.Wallet
	s.mu.Unlock()

	work, err := s.source.CurrentWork(ctx, rewardAddress)
	if err != nil {
		return nil, fmt.Errorf("get current Stratum work: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if work.WorkID == "" {
		return nil, errors.New("source returned empty work ID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.identity == nil || s.identity.Wallet != rewardAddress {
		return nil, newProtocolError(protocolInvalidRequest)
	}
	if s.currentJob != nil && s.currentJob.work.WorkID == work.WorkID {
		if s.currentJob.work != work {
			return nil, errors.New("source reused work ID with mutated immutable fields")
		}
		return nil, nil
	}

	generation := s.generation + 1
	lane, err := s.config.LaneSource.Acquire(work.WorkID, s.id)
	if err != nil {
		return nil, fmt.Errorf("acquire Stratum nonce lane: %w", err)
	}
	newJob := &job{
		id:         deriveJobID(work.WorkID, s.id, generation),
		work:       work,
		generation: generation,
		lane:       lane,
	}

	oldJob := s.currentJob
	if oldJob != nil {
		s.staleJobIDs = append(s.staleJobIDs, oldJob.id)
		if len(s.staleJobIDs) > maxStaleJobIDs {
			s.staleJobIDs = append([]string(nil), s.staleJobIDs[len(s.staleJobIDs)-maxStaleJobIDs:]...)
		}
	}
	s.currentJob = newJob
	s.generation = generation
	if oldJob != nil {
		s.config.LaneSource.Release(oldJob.work.WorkID, s.id)
	}

	return []Message{
		Notification{Method: "mining.set_difficulty", Params: []any{s.config.ShareDifficulty}},
		Notification{Method: "mining.notify", Params: notifyParams{
			JobID:             newJob.id,
			Algorithm:         work.Algorithm,
			Height:            work.Height,
			TargetHex:         work.TargetHex,
			HeaderPrefixHex:   work.HeaderPrefixHex,
			RewardAddress:     work.RewardAddress,
			Version:           work.Version,
			NetworkDifficulty: work.Difficulty,
			Lane:              lane,
			CleanJobs:         true,
		}},
	}, nil
}
