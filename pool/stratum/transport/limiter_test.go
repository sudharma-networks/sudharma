package transport

import (
	"testing"
	"time"
)

func TestTokenBucketInitialBurstAndRefill(t *testing.T) {
	start := time.Unix(1000, 0)
	bucket := newTokenBucket(2, 2, start)
	if !bucket.Allow(start) || !bucket.Allow(start) || bucket.Allow(start) {
		t.Fatal("unexpected initial burst behavior")
	}
	if !bucket.Allow(start.Add(500 * time.Millisecond)) {
		t.Fatal("one token should refill after 500ms at 2 requests/sec")
	}
	if bucket.Allow(start.Add(500 * time.Millisecond)) {
		t.Fatal("refilled token was not consumed")
	}
}

func TestTokenBucketRefillCapsAtBurst(t *testing.T) {
	start := time.Unix(2000, 0)
	bucket := newTokenBucket(4, 2, start)
	if !bucket.Allow(start) || !bucket.Allow(start) {
		t.Fatal("initial burst unavailable")
	}
	later := start.Add(10 * time.Second)
	if !bucket.Allow(later) || !bucket.Allow(later) || bucket.Allow(later) {
		t.Fatal("refill did not cap at burst")
	}
}

func TestTokenBucketBackwardTimeDoesNotMintTokens(t *testing.T) {
	start := time.Unix(3000, 0)
	bucket := newTokenBucket(1, 1, start)
	if !bucket.Allow(start) {
		t.Fatal("initial token unavailable")
	}
	if bucket.Allow(start.Add(-time.Second)) {
		t.Fatal("backward timestamp minted a token")
	}
	if !bucket.Allow(start.Add(time.Second)) {
		t.Fatal("forward elapsed time did not refill token")
	}
}
