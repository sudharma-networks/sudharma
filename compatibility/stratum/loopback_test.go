package stratumcompat

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/sudharma-networks/sudharma/pool/stratum"
	"github.com/sudharma-networks/sudharma/pool/stratum/loopback"
	"github.com/sudharma-networks/sudharma/pool/stratum/server"
)

func TestLoopbackPlaintextTranscript(t *testing.T) {
	listener, err := loopback.Listen()
	if err != nil {
		t.Fatal(err)
	}

	source := newCompatSource()
	var factoryCalls atomic.Int32
	cancel, done := startCompatServer(t, listener, newCompatFactory(t, source, &factoryCalls), server.Config{})

	conn := dialCompatTCP(t, listener.Addr().String())
	reader := bufio.NewReader(conn)

	writeRequest(t, conn, `{"id":1,"method":"mining.subscribe","params":["khushi-loopback/1.0"]}`)
	subscribe := readJSONMessage(t, reader)
	if subscribe["id"] != float64(1) || subscribe["error"] != nil {
		t.Fatalf("subscribe response = %#v", subscribe)
	}
	sessionID, ok := subscribe["result"].(string)
	if !ok || len(sessionID) != 32 {
		t.Fatalf("session ID = %#v, want 32-character string", subscribe["result"])
	}
	if _, err := hex.DecodeString(sessionID); err != nil {
		t.Fatalf("session ID is not hexadecimal: %q: %v", sessionID, err)
	}

	worker := compatWallet + ".rig_01"
	writeRequest(t, conn, fmt.Sprintf(`{"id":2,"method":"mining.authorize","params":[%q,"x"]}`, worker))
	requireResponseResult(t, reader, 2, true)
	jobID, lane := requireWorkMessages(t, reader)

	shareNonce := uint64(lane)<<32 | 1
	blockNonce := uint64(lane)<<32 | 2
	writeRequest(t, conn, fmt.Sprintf(`{"id":3,"method":"mining.submit","params":[%q,%q,%q]}`, worker, jobID, fmt.Sprintf("%016x", shareNonce)))
	requireResponseResult(t, reader, 3, string(stratum.SubmitAcceptedShare))

	writeRequest(t, conn, fmt.Sprintf(`{"id":4,"method":"mining.submit","params":[%q,%q,%q]}`, worker, jobID, fmt.Sprintf("%016x", blockNonce)))
	requireResponseResult(t, reader, 4, string(stratum.SubmitAcceptedBlock))

	writeRequest(t, conn, fmt.Sprintf(`{"id":5,"method":"mining.submit","params":[%q,%q,%q]}`, worker, jobID, fmt.Sprintf("%016x", blockNonce)))
	requireResponseResult(t, reader, 5, string(stratum.SubmitDuplicate))

	submissions := source.submitted()
	if len(submissions) != 1 {
		t.Fatalf("network candidate submissions = %d, want 1", len(submissions))
	}
	candidate := submissions[0]
	if candidate.Nonce != blockNonce || candidate.Lane != lane || candidate.JobID != jobID {
		t.Fatalf("forwarded candidate = %#v, want nonce=%016x lane=%08x job=%s", candidate, blockNonce, lane, jobID)
	}
	if candidate.Identity.Wallet != compatWallet || candidate.Identity.Worker != "rig_01" {
		t.Fatalf("forwarded identity = %#v", candidate.Identity)
	}
	if candidate.Work.WorkID != "loopback-work-1" || candidate.Work.RewardAddress != compatWallet {
		t.Fatalf("forwarded work = %#v", candidate.Work)
	}
	if got := factoryCalls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}

	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	stopCompatServer(t, cancel, done)
}

func TestLoopbackBlankPasswordAuthorizes(t *testing.T) {
	listener, err := loopback.Listen()
	if err != nil {
		t.Fatal(err)
	}

	source := newCompatSource()
	var factoryCalls atomic.Int32
	cancel, done := startCompatServer(t, listener, newCompatFactory(t, source, &factoryCalls), server.Config{})

	conn := dialCompatTCP(t, listener.Addr().String())
	reader := bufio.NewReader(conn)

	writeRequest(t, conn, `{"id":1,"method":"mining.subscribe","params":[]}`)
	subscribe := readJSONMessage(t, reader)
	if subscribe["id"] != float64(1) || subscribe["error"] != nil {
		t.Fatalf("subscribe response = %#v", subscribe)
	}

	worker := compatWallet + ".blank_pw"
	writeRequest(t, conn, fmt.Sprintf(`{"id":2,"method":"mining.authorize","params":[%q,""]}`, worker))
	requireResponseResult(t, reader, 2, true)
	_, lane := requireWorkMessages(t, reader)
	if lane != compatLaneValue {
		t.Fatalf("lane = %08x, want %08x", lane, compatLaneValue)
	}
	if got := factoryCalls.Load(); got != 1 {
		t.Fatalf("session factory calls = %d, want 1", got)
	}

	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	stopCompatServer(t, cancel, done)
}
