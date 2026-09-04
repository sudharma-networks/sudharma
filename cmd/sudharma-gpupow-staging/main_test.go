package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/pow"
)

type blackBoxChallenge struct {
	ChallengeID  string `json:"challenge_id"`
	Algorithm    string `json:"algorithm"`
	Staging      bool   `json:"staging"`
	HeaderPrefix string `json:"header_prefix"`
	Target       string `json:"target"`
	Height       uint64 `json:"height"`
	CacheNodes   uint32 `json:"cache_nodes"`
}

type blackBoxSubmission struct {
	Challenge blackBoxChallenge `json:"challenge"`
	Nonce     uint64            `json:"nonce"`
}

type blackBoxResult struct {
	Status string `json:"status"`
}

func buildStagingBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "sudharma-gpupow-staging")
	cmd := exec.Command("go", "build", "-trimpath", "-o", binary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build staging verifier: %v\n%s", err, output)
	}
	return binary
}

func reserveLoopbackAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback address: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release loopback address: %v", err)
	}
	return addr
}

func testDigestAtOrBelowTarget(digest [32]byte, target []byte) bool {
	if len(target) != len(digest) {
		return false
	}
	for i := range digest {
		if digest[i] < target[i] {
			return true
		}
		if digest[i] > target[i] {
			return false
		}
	}
	return true
}

func solveStagingChallenge(t *testing.T, challenge blackBoxChallenge) uint64 {
	t.Helper()
	header, err := hex.DecodeString(challenge.HeaderPrefix)
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	target, err := hex.DecodeString(challenge.Target)
	if err != nil {
		t.Fatalf("decode target: %v", err)
	}
	cache := pow.GPUV1BuildCache(pow.GPUV1EpochSeed(pow.GPUV1EpochForHeight(challenge.Height)), challenge.CacheNodes)
	for nonce := uint64(0); nonce < 1_000_000; nonce++ {
		digest := pow.GPUV1ReferenceDigest(header, nonce, challenge.Height, cache)
		if testDigestAtOrBelowTarget(digest, target) {
			return nonce
		}
	}
	t.Fatal("unable to solve compact staging target")
	return 0
}

func waitForChallenge(t *testing.T, endpoint string) blackBoxChallenge {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	url := endpoint + "/v1/mining/staging/challenge"
	var lastErr error
	for attempt := 0; attempt < 40; attempt++ {
		response, err := client.Get(url)
		if err == nil {
			if response.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("challenge status=%d", response.StatusCode)
				_ = response.Body.Close()
			} else {
				var challenge blackBoxChallenge
				decodeErr := json.NewDecoder(response.Body).Decode(&challenge)
				_ = response.Body.Close()
				if decodeErr == nil {
					return challenge
				}
				lastErr = decodeErr
			}
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("staging verifier did not become ready: %v", lastErr)
	return blackBoxChallenge{}
}

func submitStagingSolution(t *testing.T, endpoint string, submission blackBoxSubmission) blackBoxResult {
	t.Helper()
	body, err := json.Marshal(submission)
	if err != nil {
		t.Fatalf("marshal submission: %v", err)
	}
	response, err := http.Post(endpoint+"/v1/mining/staging/submit", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("submit staging solution: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("submit status=%d", response.StatusCode)
	}
	var result blackBoxResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode submit result: %v", err)
	}
	return result
}

func TestStagingVerifierBlackBox(t *testing.T) {
	binary := buildStagingBinary(t)
	addr := reserveLoopbackAddress(t)
	cmd := exec.Command(binary, "-listen", addr)
	var processLog bytes.Buffer
	cmd.Stdout = &processLog
	cmd.Stderr = &processLog
	if err := cmd.Start(); err != nil {
		t.Fatalf("start staging verifier: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	endpoint := "http://" + addr
	challenge := waitForChallenge(t, endpoint)
	if !challenge.Staging || challenge.Algorithm != pow.GPUV1AlgorithmID {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
	if challenge.Height != 0 || challenge.CacheNodes != 8 || challenge.ChallengeID == "" {
		t.Fatalf("unexpected compact staging parameters: %+v", challenge)
	}
	nonce := solveStagingChallenge(t, challenge)
	submission := blackBoxSubmission{Challenge: challenge, Nonce: nonce}
	if got := submitStagingSolution(t, endpoint, submission).Status; got != "accepted" {
		t.Fatalf("valid staging solution status=%q log=%s", got, processLog.String())
	}
	if got := submitStagingSolution(t, endpoint, submission).Status; got != "rejected" {
		t.Fatalf("replayed staging solution status=%q, want rejected", got)
	}
}

func TestStagingVerifierRejectsWildcardBind(t *testing.T) {
	binary := buildStagingBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "-listen", "0.0.0.0:0")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("wildcard bind was not rejected; process stayed running: %s", output)
	}
	if err == nil {
		t.Fatalf("wildcard bind unexpectedly succeeded: %s", output)
	}
}
