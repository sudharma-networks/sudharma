package p2p

import "testing"

func TestInboundHandshakeSlotsAreBounded(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < MaxConcurrentInboundHandshakes; i++ {
		if !node.tryAcquireInboundHandshake() {
			t.Fatalf("expected slot %d to be available", i)
		}
	}
	if node.tryAcquireInboundHandshake() {
		t.Fatal("expected handshake limiter to reject work above the cap")
	}
}

func TestInboundHandshakeSlotReleaseAllowsNewWork(t *testing.T) {
	node, err := NewNode("local", "127.0.0.1:0", 0, "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < MaxConcurrentInboundHandshakes; i++ {
		if !node.tryAcquireInboundHandshake() {
			t.Fatalf("expected slot %d to be available", i)
		}
	}

	node.releaseInboundHandshake()
	if !node.tryAcquireInboundHandshake() {
		t.Fatal("expected a released handshake slot to be reusable")
	}
}
