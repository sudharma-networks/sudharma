package stratum

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

const transcriptWallet = "9ccdc094489874bed888ffe4bdf9b8298f4c5131"

type transcriptSource struct {
	work        Work
	submissions []Candidate
}

func (s *transcriptSource) CurrentWork(ctx context.Context, rewardAddress string) (Work, error) {
	if err := ctx.Err(); err != nil {
		return Work{}, err
	}
	work := s.work
	work.RewardAddress = rewardAddress
	return work, nil
}

func (s *transcriptSource) Submit(ctx context.Context, candidate Candidate) (SourceResult, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.submissions = append(s.submissions, candidate)
	return SourceAccepted, nil
}

type transcriptVerifier struct {
	shareTarget   [32]byte
	networkTarget [32]byte
}

func (v transcriptVerifier) MeetsTarget(_ context.Context, _ Work, nonce uint64, target [32]byte) (bool, error) {
	switch target {
	case v.shareTarget:
		return true, nil
	case v.networkTarget:
		return uint32(nonce) == 2, nil
	default:
		return false, fmt.Errorf("unexpected transcript target %x", target)
	}
}

type transcriptLane uint32

func (l transcriptLane) Acquire(string, string) (uint32, error) { return uint32(l), nil }
func (transcriptLane) Release(string, string)                   {}

func TestOfflineStratumTranscript(t *testing.T) {
	const lane = uint32(0x01020304)
	const networkTargetHex = "0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f"

	shareTarget, err := targetFromDifficulty(4)
	if err != nil {
		t.Fatal(err)
	}
	networkTarget, err := decodeNetworkTarget(networkTargetHex)
	if err != nil {
		t.Fatal(err)
	}
	source := &transcriptSource{work: transcriptWork("work-1", 100, "aabb", networkTargetHex)}
	session, err := NewSession(
		bytes.NewReader(bytes.Repeat([]byte{0x11}, 16)),
		source,
		transcriptVerifier{shareTarget: shareTarget, networkTarget: networkTarget},
		Config{ShareDifficulty: 4, MaxDuplicateShares: 16, LaneSource: transcriptLane(lane)},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var transcript bytes.Buffer
	appendMessages := func(messages []Message) {
		for _, message := range messages {
			encoded, err := EncodeMessage(message)
			if err != nil {
				t.Fatal(err)
			}
			transcript.Write(encoded)
		}
	}
	recordRequest := func(line string) {
		transcript.WriteString(line)
		transcript.WriteByte('\n')
		messages, err := session.Handle(ctx, []byte(line))
		if err != nil {
			var protocolErr *ProtocolError
			if !errors.As(err, &protocolErr) {
				t.Fatal(err)
			}
			appendMessages([]Message{Response{ID: json.RawMessage("null"), Error: protocolErr}})
			return
		}
		appendMessages(messages)
	}
	refresh := func() {
		messages, err := session.RefreshWork(ctx)
		if err != nil {
			t.Fatal(err)
		}
		appendMessages(messages)
	}

	recordRequest(`{"id":1,"method":"mining.subscribe","params":["khushi-test/1.0"]}`)
	recordRequest(`{"id":2,"method":"mining.authorize","params":["` + transcriptWallet + `.rig_01","x"]}`)
	refresh()

	job1 := deriveJobID("work-1", session.SessionID(), 1)
	nonceShare := uint64(lane)<<32 | 1
	nonceBlock := uint64(lane)<<32 | 2
	recordRequest(fmt.Sprintf(`{"id":3,"method":"mining.submit","params":["%s.rig_01","%s","%016x"]}`, transcriptWallet, job1, nonceShare))
	recordRequest(fmt.Sprintf(`{"id":4,"method":"mining.submit","params":["%s.rig_01","%s","%016x"]}`, transcriptWallet, job1, nonceBlock))
	recordRequest(fmt.Sprintf(`{"id":5,"method":"mining.submit","params":["%s.rig_01","%s","%016x"]}`, transcriptWallet, job1, nonceBlock))

	source.work = transcriptWork("work-2", 101, "ccdd", networkTargetHex)
	refresh()
	job2 := deriveJobID("work-2", session.SessionID(), 2)
	recordRequest(fmt.Sprintf(`{"id":6,"method":"mining.submit","params":["%s.rig_01","%s","%016x"]}`, transcriptWallet, job1, uint64(lane)<<32|3))
	recordRequest(fmt.Sprintf(`{"id":7,"method":"mining.submit","params":["%s.rig_01","%s","%016x"]}`, transcriptWallet, job2, uint64(lane+1)<<32|3))
	recordRequest(`{"id":8,"method":"mining.submit","params":[`)

	if len(source.submissions) != 1 {
		t.Fatalf("network candidate submissions = %d, want 1", len(source.submissions))
	}
	if source.submissions[0].Nonce != nonceBlock {
		t.Fatalf("forwarded nonce = %016x, want %016x", source.submissions[0].Nonce, nonceBlock)
	}

	const expected = `{"id":1,"method":"mining.subscribe","params":["khushi-test/1.0"]}
{"id":1,"result":"11111111111111111111111111111111","error":null}
{"id":2,"method":"mining.authorize","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","x"]}
{"id":2,"result":true,"error":null}
{"id":null,"method":"mining.set_difficulty","params":[4]}
{"id":null,"method":"mining.notify","params":["b8834f3e1663949d9332c1b42c2b8b47200d854b0be6cd9cfb1f5d30037b2dda","sudharma-gpupow-v1",100,"0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f","aabb","9ccdc094489874bed888ffe4bdf9b8298f4c5131",2,11,16909060,true]}
{"id":3,"method":"mining.submit","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","b8834f3e1663949d9332c1b42c2b8b47200d854b0be6cd9cfb1f5d30037b2dda","0102030400000001"]}
{"id":3,"result":"accepted_share","error":null}
{"id":4,"method":"mining.submit","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","b8834f3e1663949d9332c1b42c2b8b47200d854b0be6cd9cfb1f5d30037b2dda","0102030400000002"]}
{"id":4,"result":"accepted_block","error":null}
{"id":5,"method":"mining.submit","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","b8834f3e1663949d9332c1b42c2b8b47200d854b0be6cd9cfb1f5d30037b2dda","0102030400000002"]}
{"id":5,"result":"duplicate","error":null}
{"id":null,"method":"mining.set_difficulty","params":[4]}
{"id":null,"method":"mining.notify","params":["8d5c536ad1b2592b4a9937743eadbea48abd82422850110b9a3db0e590d99504","sudharma-gpupow-v1",101,"0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f","ccdd","9ccdc094489874bed888ffe4bdf9b8298f4c5131",2,11,16909060,true]}
{"id":6,"method":"mining.submit","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","b8834f3e1663949d9332c1b42c2b8b47200d854b0be6cd9cfb1f5d30037b2dda","0102030400000003"]}
{"id":6,"result":"stale","error":null}
{"id":7,"method":"mining.submit","params":["9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01","8d5c536ad1b2592b4a9937743eadbea48abd82422850110b9a3db0e590d99504","0102030500000003"]}
{"id":7,"result":"invalid","error":null}
{"id":8,"method":"mining.submit","params":[
{"id":null,"result":null,"error":{"code":-32700,"message":"parse error"}}
`
	if got := transcript.String(); got != expected {
		t.Fatalf("offline Stratum transcript mismatch\n--- got ---\n%s--- want ---\n%s", got, expected)
	}
}

func transcriptWork(workID string, height uint64, headerPrefixHex, targetHex string) Work {
	return Work{
		WorkID:          workID,
		Algorithm:       "sudharma-gpupow-v1",
		TargetHex:       targetHex,
		HeaderPrefixHex: headerPrefixHex,
		RewardAddress:   transcriptWallet,
		Version:         2,
		Height:          height,
		Difficulty:      11,
	}
}
