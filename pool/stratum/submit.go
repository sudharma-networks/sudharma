package stratum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

const maxJobIDBytes = 128

const (
	SubmitAcceptedShare SubmitStatus = "accepted_share"
	SubmitAcceptedBlock SubmitStatus = "accepted_block"
	SubmitInvalid       SubmitStatus = "invalid"
	SubmitDuplicate     SubmitStatus = "duplicate"
	SubmitStale         SubmitStatus = "stale"
	SubmitMutated       SubmitStatus = "mutated"
)

type shareKey struct {
	jobID string
	nonce uint64
}

func (s *Session) handleSubmit(ctx context.Context, request Request) ([]Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	workerValue, jobID, nonce, err := decodeSubmitParams(request.Params)
	if err != nil {
		return nil, err
	}
	identity, err := ParseWorkerIdentity(workerValue)
	if err != nil {
		return submitResponse(request.ID, SubmitInvalid), nil
	}

	s.mu.Lock()
	if s.identity == nil || s.currentJob == nil {
		s.mu.Unlock()
		return nil, newProtocolError(protocolInvalidRequest)
	}
	if *s.identity != identity {
		s.mu.Unlock()
		return submitResponse(request.ID, SubmitInvalid), nil
	}
	if jobID != s.currentJob.id {
		stale := containsStaleJobID(s.staleJobIDs, jobID)
		s.mu.Unlock()
		if stale {
			return submitResponse(request.ID, SubmitStale), nil
		}
		return submitResponse(request.ID, SubmitInvalid), nil
	}
	if uint32(nonce>>32) != s.currentJob.lane {
		s.mu.Unlock()
		return submitResponse(request.ID, SubmitInvalid), nil
	}

	key := shareKey{jobID: jobID, nonce: nonce}
	if s.duplicateShares == nil {
		s.duplicateShares = make(map[shareKey]struct{})
	}
	if _, exists := s.duplicateShares[key]; exists {
		s.mu.Unlock()
		return submitResponse(request.ID, SubmitDuplicate), nil
	}
	if len(s.duplicateShares) >= s.config.MaxDuplicateShares {
		s.mu.Unlock()
		return nil, errors.New("stratum duplicate share limit reached")
	}
	s.duplicateShares[key] = struct{}{}

	current := *s.currentJob
	authorized := *s.identity
	shareDifficulty := s.config.ShareDifficulty
	s.mu.Unlock()

	shareTarget, err := targetFromDifficulty(shareDifficulty)
	if err != nil {
		return nil, err
	}
	networkTarget, err := decodeNetworkTarget(current.work.TargetHex)
	if err != nil {
		return nil, err
	}

	meetsShare, err := s.verifier.MeetsTarget(ctx, current.work, nonce, shareTarget)
	if err != nil {
		s.removeDuplicateAfterVerifierError(key)
		return nil, fmt.Errorf("verify Stratum share target: %w", err)
	}
	if !meetsShare {
		return submitResponse(request.ID, SubmitInvalid), nil
	}

	meetsNetwork, err := s.verifier.MeetsTarget(ctx, current.work, nonce, networkTarget)
	if err != nil {
		s.removeDuplicateAfterVerifierError(key)
		return nil, fmt.Errorf("verify Stratum network target: %w", err)
	}

	s.mu.Lock()
	stillCurrent := s.currentJob != nil && s.currentJob.id == current.id && s.currentJob.generation == current.generation
	s.mu.Unlock()
	if !stillCurrent {
		return submitResponse(request.ID, SubmitStale), nil
	}

	if !meetsNetwork {
		return submitResponse(request.ID, SubmitAcceptedShare), nil
	}

	result, err := s.source.Submit(ctx, Candidate{
		Work:     current.work,
		JobID:    current.id,
		Identity: authorized,
		Lane:     current.lane,
		Nonce:    nonce,
	})
	if err != nil {
		return nil, fmt.Errorf("submit Stratum network candidate: %w", err)
	}

	status, err := mapSourceResult(result)
	if err != nil {
		return nil, err
	}
	return submitResponse(request.ID, status), nil
}

func decodeSubmitParams(raw json.RawMessage) (string, string, uint64, error) {
	var params []json.RawMessage
	if err := json.Unmarshal(raw, &params); err != nil || len(params) != 3 {
		return "", "", 0, newProtocolError(protocolInvalidParams)
	}

	var worker, jobID, nonceHex string
	if err := json.Unmarshal(params[0], &worker); err != nil {
		return "", "", 0, newProtocolError(protocolInvalidParams)
	}
	if err := json.Unmarshal(params[1], &jobID); err != nil || len(jobID) == 0 || len(jobID) > maxJobIDBytes {
		return "", "", 0, newProtocolError(protocolInvalidParams)
	}
	if err := json.Unmarshal(params[2], &nonceHex); err != nil {
		return "", "", 0, newProtocolError(protocolInvalidParams)
	}
	if len(nonceHex) == 0 || len(nonceHex) > 16 {
		return "", "", 0, newProtocolError(protocolInvalidParams)
	}
	for i := 0; i < len(nonceHex); i++ {
		c := nonceHex[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", "", 0, newProtocolError(protocolInvalidParams)
		}
	}
	nonce, err := strconv.ParseUint(nonceHex, 16, 64)
	if err != nil {
		return "", "", 0, newProtocolError(protocolInvalidParams)
	}
	return worker, jobID, nonce, nil
}

func targetFromDifficulty(difficulty uint32) ([32]byte, error) {
	var target [32]byte
	if difficulty == 0 {
		return target, errors.New("stratum share difficulty must be positive")
	}
	maxHash := new(big.Int).Lsh(big.NewInt(1), 256)
	maxHash.Sub(maxHash, big.NewInt(1))
	maxHash.Div(maxHash, new(big.Int).SetUint64(uint64(difficulty)))
	maxHash.FillBytes(target[:])
	return target, nil
}

func decodeNetworkTarget(value string) ([32]byte, error) {
	var target [32]byte
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != len(target) {
		return target, errors.New("invalid Stratum network target")
	}
	copy(target[:], decoded)
	return target, nil
}

func containsStaleJobID(stale []string, jobID string) bool {
	for _, id := range stale {
		if id == jobID {
			return true
		}
	}
	return false
}

func (s *Session) removeDuplicateAfterVerifierError(key shareKey) {
	s.mu.Lock()
	delete(s.duplicateShares, key)
	s.mu.Unlock()
}

func mapSourceResult(result SourceResult) (SubmitStatus, error) {
	switch result {
	case SourceAccepted:
		return SubmitAcceptedBlock, nil
	case SourceInvalid:
		return SubmitInvalid, nil
	case SourceStale:
		return SubmitStale, nil
	case SourceMutated:
		return SubmitMutated, nil
	default:
		return "", fmt.Errorf("unknown Stratum source result %q", result)
	}
}

func submitResponse(id json.RawMessage, status SubmitStatus) []Message {
	return []Message{Response{ID: id, Result: status}}
}
