package publicrpc

import "testing"

func TestMatchRouteAllowlist(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   RouteKind
	}{
		{"GET", "/health", RouteHealth},
		{"GET", "/ready", RouteReady},
		{"GET", "/v1/status", RouteStatus},
		{"GET", "/v1/accounts/alice", RouteAccount},
		{"POST", "/v1/transactions", RouteSubmitTransaction},
		{"GET", "/v1/transactions/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RouteTransactionStatus},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got, _, err := MatchRoute(tt.method, tt.path)
			if err != nil {
				t.Fatalf("MatchRoute() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MatchRoute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchRouteRejectsForbiddenAndMalformedPaths(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
	}{
		{"GET", "/metrics"},
		{"GET", "/v1/blocks/0"},
		{"GET", "/v1/mempool"},
		{"POST", "/health"},
		{"GET", "/v1/accounts/"},
		{"GET", "/v1/accounts/a/b"},
		{"GET", "/v1/accounts/%2e%2e%2fmetrics"},
		{"GET", "/v1/transactions/not-a-transaction-id"},
		{"GET", "/v1/transactions/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{"GET", "/v1/transactions/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			if _, _, err := MatchRoute(tc.method, tc.path); err == nil {
				t.Fatalf("MatchRoute(%q, %q) unexpectedly succeeded", tc.method, tc.path)
			}
		})
	}
}

func TestValidTransactionID(t *testing.T) {
	if !ValidTransactionID("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef") {
		t.Fatal("expected lowercase 64-hex transaction id to be valid")
	}
	for _, bad := range []string{"", "abc", "g123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"} {
		if ValidTransactionID(bad) {
			t.Fatalf("ValidTransactionID(%q) = true", bad)
		}
	}
}
