package rpc

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/sudharma-networks/sudharma/pow"
)

const (
	miningStagingMaxOutstanding = 16
	miningStagingChallengeTTL    = 5 * time.Minute
)

var (
	miningStagingChallengeDomain = []byte("SUDHARMA-GPU-POW-V1-STAGING-CHALLENGE\x00")
	miningStagingNow             = time.Now
)

// MiningStagingChallenge is deliberately not a block template. It exists only
// to prove that a physical GPU search result agrees with a Go verifier before
// production cache/DAG policy or Version-2 consensus activation is enabled.
type MiningStagingChallenge struct {
	ChallengeID     string `json:"challenge_id"`
	Algorithm       string `json:"algorithm"`
	Height          uint64 `json:"height"`
	HeaderPrefixHex string `json:"header_prefix"`
	TargetHex       string `json:"target"`
	CacheNodes      uint32 `json:"cache_nodes"`
	Staging         bool   `json:"staging"`
}

type MiningStagingSolution struct {
	Challenge MiningStagingChallenge `json:"challenge"`
	Nonce     uint64                 `json:"nonce"`
}

type MiningStagingVerifier func(challenge MiningStagingChallenge, nonce uint64) bool

type miningStagingActiveChallenge struct {
	challenge MiningStagingChallenge
	issuedAt  time.Time
}

type MiningStagingService struct {
	mu       sync.Mutex
	verifier MiningStagingVerifier
	active   map[string]miningStagingActiveChallenge
	order    []string
}

func NewMiningStagingService(verifier MiningStagingVerifier) *MiningStagingService {
	return &MiningStagingService{
		verifier: verifier,
		active:   make(map[string]miningStagingActiveChallenge),
	}
}

// Issue creates explicit non-consensus work. cacheNodes is mandatory on every
// challenge so staging can never silently select or freeze a production cache
// policy. target must be exactly one 256-bit big-endian value.
func (s *MiningStagingService) Issue(headerPrefix []byte, height uint64, cacheNodes uint32, target []byte) (MiningStagingChallenge, error) {
	if s == nil {
		return MiningStagingChallenge{}, fmt.Errorf("staging service is nil")
	}
	if len(headerPrefix) == 0 {
		return MiningStagingChallenge{}, fmt.Errorf("staging header prefix is required")
	}
	if cacheNodes == 0 {
		return MiningStagingChallenge{}, fmt.Errorf("staging cache node count must be positive")
	}
	if len(target) != 32 {
		return MiningStagingChallenge{}, fmt.Errorf("staging target must be exactly 32 bytes")
	}

	var heightBytes [8]byte
	binary.BigEndian.PutUint64(heightBytes[:], height)
	var cacheBytes [4]byte
	binary.BigEndian.PutUint32(cacheBytes[:], cacheNodes)

	idInput := make([]byte, 0, len(miningStagingChallengeDomain)+len(pow.GPUV1AlgorithmID)+len(heightBytes)+len(cacheBytes)+len(target)+len(headerPrefix))
	idInput = append(idInput, miningStagingChallengeDomain...)
	idInput = append(idInput, pow.GPUV1AlgorithmID...)
	idInput = append(idInput, heightBytes[:]...)
	idInput = append(idInput, cacheBytes[:]...)
	idInput = append(idInput, target...)
	idInput = append(idInput, headerPrefix...)
	id := sha256.Sum256(idInput)

	challenge := MiningStagingChallenge{
		ChallengeID:     hex.EncodeToString(id[:]),
		Algorithm:       pow.GPUV1AlgorithmID,
		Height:          height,
		HeaderPrefixHex: hex.EncodeToString(headerPrefix),
		TargetHex:       hex.EncodeToString(target),
		CacheNodes:      cacheNodes,
		Staging:         true,
	}

	now := miningStagingNow()
	s.mu.Lock()
	s.pruneActiveLocked(now)
	if _, exists := s.active[challenge.ChallengeID]; !exists {
		for len(s.active) >= miningStagingMaxOutstanding && len(s.order) > 0 {
			oldest := s.order[0]
			s.order = s.order[1:]
			delete(s.active, oldest)
		}
		s.order = append(s.order, challenge.ChallengeID)
	}
	s.active[challenge.ChallengeID] = miningStagingActiveChallenge{challenge: challenge, issuedAt: now}
	s.mu.Unlock()
	return challenge, nil
}

func (s *MiningStagingService) Submit(solution MiningStagingSolution) MiningSubmitResult {
	if s == nil {
		return MiningSubmitResult{Status: MiningSubmitStale}
	}

	now := miningStagingNow()
	s.mu.Lock()
	s.pruneActiveLocked(now)
	entry, ok := s.active[solution.Challenge.ChallengeID]
	verifier := s.verifier
	s.mu.Unlock()
	if !ok {
		return MiningSubmitResult{Status: MiningSubmitStale}
	}

	active := entry.challenge
	if !stagingChallengeMatches(solution.Challenge, active) {
		return MiningSubmitResult{Status: MiningSubmitMutated}
	}
	if verifier == nil || !verifier(active, solution.Nonce) {
		return MiningSubmitResult{Status: MiningSubmitInvalid}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneActiveLocked(miningStagingNow())
	current, ok := s.active[active.ChallengeID]
	if !ok || !stagingChallengeMatches(current.challenge, active) {
		return MiningSubmitResult{Status: MiningSubmitStale}
	}
	delete(s.active, active.ChallengeID)
	return MiningSubmitResult{Status: MiningSubmitAccepted}
}

func (s *MiningStagingService) pruneActiveLocked(now time.Time) {
	if len(s.order) == 0 {
		return
	}
	kept := s.order[:0]
	for _, id := range s.order {
		entry, ok := s.active[id]
		if !ok {
			continue
		}
		if now.Sub(entry.issuedAt) > miningStagingChallengeTTL {
			delete(s.active, id)
			continue
		}
		kept = append(kept, id)
	}
	s.order = kept
}

func stagingChallengeMatches(got, want MiningStagingChallenge) bool {
	return got.ChallengeID == want.ChallengeID &&
		got.Algorithm == want.Algorithm &&
		got.Height == want.Height &&
		got.HeaderPrefixHex == want.HeaderPrefixHex &&
		got.TargetHex == want.TargetHex &&
		got.CacheNodes == want.CacheNodes &&
		got.Staging == want.Staging
}
