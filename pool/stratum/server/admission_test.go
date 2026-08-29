package server

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

type fallbackAddr string

func (a fallbackAddr) Network() string { return "test" }
func (a fallbackAddr) String() string  { return string(a) }

func TestSourceKeyNormalizesTCPAddresses(t *testing.T) {
	tests := []struct {
		name string
		addr net.Addr
		want string
	}{
		{
			name: "ipv4 first port",
			addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1000},
			want: "127.0.0.1",
		},
		{
			name: "ipv4 second port",
			addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 2000},
			want: "127.0.0.1",
		},
		{
			name: "ipv4 mapped ipv6",
			addr: &net.TCPAddr{IP: net.ParseIP("::ffff:127.0.0.1"), Port: 3000},
			want: "127.0.0.1",
		},
		{
			name: "native ipv6",
			addr: &net.TCPAddr{IP: net.ParseIP("2001:db8::1"), Port: 4000},
			want: "2001:db8::1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sourceKey(test.addr); got != test.want {
				t.Fatalf("sourceKey(%v) = %q, want %q", test.addr, got, test.want)
			}
		})
	}
}

func TestSourceKeyFallsBackToAddressString(t *testing.T) {
	const want = "opaque-source"
	if got := sourceKey(fallbackAddr(want)); got != want {
		t.Fatalf("sourceKey fallback = %q, want %q", got, want)
	}
}

func TestAdmissionEnforcesPerSourceAndGlobalLimits(t *testing.T) {
	a := newAdmission(2, 1)
	if !a.Acquire("ip-a") {
		t.Fatal("first admission rejected")
	}
	if a.Acquire("ip-a") {
		t.Fatal("per-source limit bypassed")
	}
	if !a.Acquire("ip-b") {
		t.Fatal("different source rejected")
	}
	if a.Acquire("ip-c") {
		t.Fatal("global limit bypassed")
	}
	a.Release("ip-a")
	if !a.Acquire("ip-c") {
		t.Fatal("released slot not reusable")
	}
}

func TestAdmissionReleaseCannotUnderflow(t *testing.T) {
	a := newAdmission(1, 1)
	a.Release("missing")
	if !a.Acquire("ip-a") {
		t.Fatal("release of missing source corrupted capacity")
	}
	a.Release("ip-a")
	a.Release("ip-a")
	if !a.Acquire("ip-b") {
		t.Fatal("duplicate release corrupted capacity")
	}
}

func TestAdmissionConcurrentNeverExceedsLimits(t *testing.T) {
	a := newAdmission(3, 2)
	start := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	var current atomic.Int32
	var maximum atomic.Int32

	for i := 0; i < 32; i++ {
		key := "ip-a"
		if i%2 == 1 {
			key = "ip-b"
		}
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			<-start
			if !a.Acquire(key) {
				return
			}
			active := current.Add(1)
			for {
				seen := maximum.Load()
				if active <= seen || maximum.CompareAndSwap(seen, active) {
					break
				}
			}
			<-release
			current.Add(-1)
			a.Release(key)
		}(key)
	}

	close(start)
	for maximum.Load() == 0 {
	}
	close(release)
	wg.Wait()

	if got := maximum.Load(); got > 3 {
		t.Fatalf("maximum concurrent admissions = %d, want <= 3", got)
	}
	if got := current.Load(); got != 0 {
		t.Fatalf("active admissions after release = %d, want 0", got)
	}
}
