package gpupowv1

import (
	"os"
	"strings"
	"testing"
)

func TestStagingDeploymentIsolatedFromSeedRPCAndLiveMining(t *testing.T) {
	service, err := os.ReadFile("../../deployment/staging/sudharma-gpupow-staging.service")
	if err != nil {
		t.Fatal(err)
	}
	serviceText := string(service)
	for _, token := range []string{
		"ExecStart=/usr/local/bin/sudharma-gpupow-staging -listen 127.0.0.1:28646",
		"NoNewPrivileges=true",
		"PrivateTmp=true",
		"ProtectSystem=strict",
		"ProtectHome=true",
		"RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX",
	} {
		if !strings.Contains(serviceText, token) {
			t.Fatalf("staging service missing hardening token %q", token)
		}
	}
	for _, forbidden := range []string{"sudharma-rpcd", "28545", "28444"} {
		if strings.Contains(serviceText, forbidden) {
			t.Fatalf("staging service must not reference seed-node surface %q", forbidden)
		}
	}

	nginx, err := os.ReadFile("../../deployment/staging/nginx-staging.example.conf")
	if err != nil {
		t.Fatal(err)
	}
	nginxText := string(nginx)
	for _, token := range []string{
		"location = /v1/mining/staging/challenge",
		"location = /v1/mining/staging/submit",
		"proxy_pass http://127.0.0.1:28646",
		"limit_req_zone",
		"client_max_body_size 64k",
		"location /",
		"return 404",
	} {
		if !strings.Contains(nginxText, token) {
			t.Fatalf("staging nginx config missing isolation token %q", token)
		}
	}
	for _, forbidden := range []string{
		"/v1/mining/work",
		"/v1/mining/submit",
		"28545",
		"28444",
	} {
		if strings.Contains(nginxText, forbidden) {
			t.Fatalf("staging nginx must not expose live/seed surface %q", forbidden)
		}
	}
}
