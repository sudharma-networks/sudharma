package publicrpc

import (
	"os"
	"strings"
	"testing"
)

func TestNginxWalletProxyPolicy(t *testing.T) {
	b, err := os.ReadFile("nginx-wallet-proxy.conf")
	if err != nil {
		t.Fatalf("read nginx-wallet-proxy.conf: %v", err)
	}
	cfg := string(b)

	required := []string{
		"listen 29100",
		"proxy_pass http://127.0.0.1:28545",
		"client_max_body_size 1m",
		"proxy_connect_timeout 2s",
		"proxy_read_timeout 8s",
		"proxy_send_timeout 8s",
		"Cache-Control \"no-store\"",
		"location = /health",
		"location = /ready",
		"location = /v1/status",
		"location ~ ^/v1/accounts/",
		"location = /v1/transactions",
		"location ~ ^/v1/transactions/",
		"location /",
		"return 404",
	}
	for _, needle := range required {
		if !strings.Contains(cfg, needle) {
			t.Errorf("missing required policy fragment %q", needle)
		}
	}

	for _, forbidden := range []string{"location /metrics", "location /v1/blocks", "location /v1/mempool"} {
		if strings.Contains(cfg, forbidden) {
			t.Errorf("forbidden route exposed: %q", forbidden)
		}
	}
}
