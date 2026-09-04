package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"

	"github.com/sudharma-networks/sudharma/params"
	"github.com/sudharma-networks/sudharma/pow"
)

type rehearsalStatus struct {
	Mode           string `json:"mode"`
	Network        string `json:"network"`
	ChainHeight    uint64 `json:"chain_height"`
	AcceptedBlocks uint64 `json:"accepted_blocks"`
	TargetBlocks   uint64 `json:"target_blocks"`
	IssuedSupply   uint64 `json:"issued_supply"`
	Completed      bool   `json:"completed"`
}

func solveRehearsalChallenge(t *testing.T, challenge blackBoxChallenge, cache []pow.GPUV1CacheNode) uint64 {
	t.Helper()
	header, err := hex.DecodeString(challenge.HeaderPrefix)
	if err != nil {
		t.Fatalf("decode rehearsal header: %v", err)
	}
	target, err := hex.DecodeString(challenge.Target)
	if err != nil {
		t.Fatalf("decode rehearsal target: %v", err)
	}
	for nonce := uint64(0); nonce < 1_000_000; nonce++ {
		digest := pow.GPUV1ReferenceDigest(header, nonce, challenge.Height, cache)
		if testDigestAtOrBelowTarget(digest, target) {
			return nonce
		}
	}
	t.Fatal("unable to solve mainnet rehearsal target")
	return 0
}

func waitForRehearsalChallenge(t *testing.T, endpoint string) blackBoxChallenge {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	url := endpoint + "/v1/mining/staging/challenge"
	var lastErr error
	for attempt := 0; attempt < 80; attempt++ {
		response, err := client.Get(url)
		if err == nil {
			if response.StatusCode == http.StatusOK {
				var challenge blackBoxChallenge
				decodeErr := json.NewDecoder(response.Body).Decode(&challenge)
				_ = response.Body.Close()
				if decodeErr == nil {
					return challenge
				}
				lastErr = decodeErr
			} else {
				lastErr = fmt.Errorf("challenge status=%d", response.StatusCode)
				_ = response.Body.Close()
			}
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("mainnet rehearsal verifier did not become ready: %v", lastErr)
	return blackBoxChallenge{}
}

func fetchRehearsalStatus(t *testing.T, endpoint string) rehearsalStatus {
	t.Helper()
	response, err := http.Get(endpoint + "/v1/mining/staging/status")
	if err != nil {
		t.Fatalf("fetch rehearsal status: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("rehearsal status code=%d", response.StatusCode)
	}
	var status rehearsalStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("decode rehearsal status: %v", err)
	}
	return status
}

func TestMainnetRehearsalMinesAndAcceptsAtLeast25KhushiBlocks(t *testing.T) {
	if params.MainnetLaunchAuthorized || params.MainnetMiningAuthorized || params.MainnetGenesisTimestamp != 0 {
		t.Fatal("public mainnet gates must remain closed during rehearsal")
	}
	if params.GPUV1MainnetActivationHeight != params.GPUV1ActivationDisabled {
		t.Fatal("public mainnet GPU activation must remain disabled during rehearsal")
	}

	const blocks = uint64(25)
	binary := buildStagingBinary(t)
	addr := reserveLoopbackAddress(t)
	cmd := exec.Command(binary,
		"-listen", addr,
		"-mainnet-rehearsal",
		"-rehearsal-blocks", fmt.Sprint(blocks),
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mainnet rehearsal verifier: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	endpoint := "http://" + addr
	var cache []pow.GPUV1CacheNode
	var cacheEpoch uint64
	for height := uint64(1); height <= blocks; height++ {
		challenge := waitForRehearsalChallenge(t, endpoint)
		if challenge.Height != height {
			t.Fatalf("challenge height=%d want=%d", challenge.Height, height)
		}
		if challenge.CacheNodes != pow.GPUV1ProductionCacheNodes {
			t.Fatalf("challenge cache_nodes=%d want production=%d", challenge.CacheNodes, pow.GPUV1ProductionCacheNodes)
		}
		epoch := pow.GPUV1EpochForHeight(height)
		if cache == nil || epoch != cacheEpoch {
			cache = pow.GPUV1BuildCache(pow.GPUV1EpochSeed(epoch), pow.GPUV1ProductionCacheNodes)
			cacheEpoch = epoch
		}
		nonce := solveRehearsalChallenge(t, challenge, cache)
		result := submitStagingSolution(t, endpoint, blackBoxSubmission{Challenge: challenge, Nonce: nonce})
		if result.Status != "accepted" {
			t.Fatalf("height %d solution status=%q", height, result.Status)
		}
	}

	status := fetchRehearsalStatus(t, endpoint)
	if status.Mode != "mainnet-rehearsal" || status.Network != string(params.NetworkMainnet) {
		t.Fatalf("unexpected rehearsal identity: %+v", status)
	}
	if status.ChainHeight != blocks || status.AcceptedBlocks != blocks || status.TargetBlocks != blocks || !status.Completed {
		t.Fatalf("unexpected rehearsal completion status: %+v", status)
	}
	if status.IssuedSupply == 0 || status.IssuedSupply > params.MainnetMaxSupply {
		t.Fatalf("unexpected mainnet issued supply after rehearsal: %d", status.IssuedSupply)
	}
}
