package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	gpupowv1 "github.com/sudharma-networks/sudharma/compatibility/gpupowv1"
	"github.com/sudharma-networks/sudharma/pow"
	"github.com/sudharma-networks/sudharma/rpc"
)

const (
	stagingHeight     uint64 = 0
	stagingCacheNodes uint32 = 8
)

var (
	stagingHeader = []byte("physical-gpu-staging-gate")
	stagingTarget = [32]byte{0x0f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

func digestAtOrBelowTarget(digest [32]byte, target []byte) bool {
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

// verifyStagingSolution is deliberately independent from the consensus/reference
// pow implementation. It is used only for the pre-activation physical-GPU
// interoperability gate and accepts only the fixed height-0/eight-cache-node
// staging contract already covered by the locked vector corpus.
func verifyStagingSolution(challenge rpc.MiningStagingChallenge, nonce uint64) bool {
	if !challenge.Staging || challenge.Algorithm != pow.GPUV1AlgorithmID {
		return false
	}
	if challenge.Height != stagingHeight || challenge.CacheNodes != stagingCacheNodes {
		return false
	}
	header, err := hex.DecodeString(challenge.HeaderPrefixHex)
	if err != nil || len(header) == 0 {
		return false
	}
	target, err := hex.DecodeString(challenge.TargetHex)
	if err != nil || len(target) != 32 {
		return false
	}
	digest := gpupowv1.Digest(header, nonce, challenge.Height, challenge.CacheNodes)
	return digestAtOrBelowTarget(digest, target)
}

func stagingChallengeProvider() ([]byte, uint64, uint32, []byte, error) {
	header := append([]byte(nil), stagingHeader...)
	target := append([]byte(nil), stagingTarget[:]...)
	return header, stagingHeight, stagingCacheNodes, target, nil
}

func main() {
	listen := flag.String("listen", "127.0.0.1:28646", "staging-only HTTP listen address")
	flag.Parse()

	service := rpc.NewMiningStagingService(verifyStagingSolution)
	api := rpc.NewMiningStagingHTTPAPI(service, stagingChallengeProvider, nil)
	server := &http.Server{
		Addr:              *listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("Sudharma GPU-PoW v1 staging verifier listening on %s (non-consensus; height=%d cache_nodes=%d)", *listen, stagingHeight, stagingCacheNodes)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(fmt.Errorf("staging verifier server: %w", err))
	}
}
