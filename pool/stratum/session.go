package stratum

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

const defaultMaxDuplicateShares = 65_536

type LaneSource interface {
	Acquire(workID, sessionID string) (uint32, error)
	Release(workID, sessionID string)
}

type Config struct {
	ShareDifficulty    uint32
	MaxDuplicateShares int
	Entropy            io.Reader
	LaneSource         LaneSource
}

type Session struct {
	mu sync.Mutex

	id       string
	source   WorkSource
	verifier ShareVerifier
	config   Config

	subscribed bool
	identity   *WorkerIdentity

	generation      uint64
	currentJob      *job
	staleJobIDs     []string
	duplicateShares map[shareKey]struct{}
}

func NewSession(entropy io.Reader, source WorkSource, verifier ShareVerifier, config Config) (*Session, error) {
	if source == nil {
		return nil, errors.New("stratum work source is required")
	}
	if verifier == nil {
		return nil, errors.New("stratum share verifier is required")
	}
	if config.LaneSource == nil {
		return nil, errors.New("stratum lane source is required")
	}
	if config.ShareDifficulty == 0 {
		return nil, errors.New("stratum share difficulty must be positive")
	}
	if config.MaxDuplicateShares == 0 {
		config.MaxDuplicateShares = defaultMaxDuplicateShares
	}
	if config.MaxDuplicateShares < 0 || config.MaxDuplicateShares > defaultMaxDuplicateShares {
		return nil, fmt.Errorf("stratum duplicate share limit must be between 1 and %d", defaultMaxDuplicateShares)
	}

	if entropy == nil {
		entropy = config.Entropy
	}
	if entropy == nil {
		entropy = rand.Reader
	}
	var sessionEntropy [16]byte
	if _, err := io.ReadFull(entropy, sessionEntropy[:]); err != nil {
		return nil, fmt.Errorf("read stratum session entropy: %w", err)
	}

	return &Session{
		id:       hex.EncodeToString(sessionEntropy[:]),
		source:   source,
		verifier: verifier,
		config:   config,
	}, nil
}

func (s *Session) SessionID() string {
	return s.id
}

func (s *Session) Handle(ctx context.Context, line []byte) ([]Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request, err := DecodeRequest(line)
	if err != nil {
		return nil, err
	}

	switch request.Method {
	case "mining.subscribe":
		return s.handleSubscribe(request)
	case "mining.authorize":
		return s.handleAuthorize(request)
	case "mining.submit":
		return s.handleSubmit(ctx, request)
	default:
		return nil, newProtocolError(protocolMethodNotFound)
	}
}

func (s *Session) handleSubscribe(request Request) ([]Message, error) {
	var params []json.RawMessage
	if err := json.Unmarshal(request.Params, &params); err != nil || len(params) > 1 {
		return nil, newProtocolError(protocolInvalidParams)
	}
	if len(params) == 1 {
		var agent string
		if err := json.Unmarshal(params[0], &agent); err != nil {
			return nil, newProtocolError(protocolInvalidParams)
		}
	}

	s.mu.Lock()
	s.subscribed = true
	s.mu.Unlock()

	return []Message{Response{ID: request.ID, Result: s.id}}, nil
}

func (s *Session) handleAuthorize(request Request) ([]Message, error) {
	s.mu.Lock()
	if !s.subscribed {
		s.mu.Unlock()
		return nil, newProtocolError(protocolInvalidRequest)
	}
	s.mu.Unlock()

	var params []json.RawMessage
	if err := json.Unmarshal(request.Params, &params); err != nil || len(params) != 2 {
		return nil, newProtocolError(protocolInvalidParams)
	}
	var username, password string
	if err := json.Unmarshal(params[0], &username); err != nil {
		return nil, newProtocolError(protocolInvalidParams)
	}
	if err := json.Unmarshal(params[1], &password); err != nil {
		return nil, newProtocolError(protocolInvalidParams)
	}
	if password != "" && password != "x" {
		return nil, newProtocolError(protocolInvalidParams)
	}
	identity, err := ParseWorkerIdentity(username)
	if err != nil {
		return nil, newProtocolError(protocolInvalidParams)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.subscribed {
		return nil, newProtocolError(protocolInvalidRequest)
	}
	if s.identity != nil && *s.identity != identity {
		return nil, newProtocolError(protocolInvalidRequest)
	}
	if s.identity == nil {
		stored := identity
		s.identity = &stored
	}

	return []Message{Response{ID: request.ID, Result: true}}, nil
}
