#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="${CONFIG_PATH:-/etc/nginx/sites-enabled/sudharma-wallet-proxy}"
PRIVATE_IP="${PRIVATE_IP:-}"
BEGIN_MARKER='# BEGIN SUDHARMA EXPLORER READ-ONLY ALLOWLIST'
END_MARKER='# END SUDHARMA EXPLORER READ-ONLY ALLOWLIST'

if [[ -z "$PRIVATE_IP" ]]; then
  echo 'PRIVATE_IP is required' >&2
  exit 1
fi
if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "nginx wallet proxy config not found: $CONFIG_PATH" >&2
  exit 1
fi

backup="$(mktemp "${CONFIG_PATH}.explorer-backup.XXXXXX")"
work="$(mktemp "${CONFIG_PATH}.explorer-work.XXXXXX")"
cleanup() {
  rm -f "$backup" "$work"
}
trap cleanup EXIT INT TERM
cp -p "$CONFIG_PATH" "$backup"

restore_config() {
  cp -p "$backup" "$CONFIG_PATH"
}

if ! grep -Fq "$BEGIN_MARKER" "$CONFIG_PATH"; then
  python3 - "$CONFIG_PATH" "$work" <<'PY'
import pathlib
import sys

source = pathlib.Path(sys.argv[1])
target = pathlib.Path(sys.argv[2])
text = source.read_text()
needle = "    location / {\n        return 404;\n    }"
if text.count(needle) != 1:
    raise SystemExit("expected exactly one nginx catch-all 404 location")

block = r'''# BEGIN SUDHARMA EXPLORER READ-ONLY ALLOWLIST
location = /v1/explorer/status {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545/v1/explorer/status;
}

location = /v1/explorer/blocks {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545/v1/explorer/blocks;
}

location ~ ^/v1/explorer/blocks/(?:[0-9]+|[0-9a-f]{64})$ {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545;
}

location = /v1/explorer/transactions {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545/v1/explorer/transactions;
}

location ~ ^/v1/explorer/transactions/[0-9a-f]{64}$ {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545;
}

location ~ ^/v1/explorer/addresses/[0-9a-f]{40}$ {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545;
}

location = /v1/explorer/search {
    limit_except GET { deny all; }
    proxy_http_version 1.1;
    proxy_set_header Host localhost;
    proxy_connect_timeout 2s;
    proxy_read_timeout 5s;
    proxy_pass http://127.0.0.1:28545/v1/explorer/search;
}
# END SUDHARMA EXPLORER READ-ONLY ALLOWLIST

'''
target.write_text(text.replace(needle, block + needle))
PY
  cp -p "$work" "$CONFIG_PATH"
fi

if ! nginx -t; then
  restore_config
  echo 'nginx validation failed; restored previous wallet proxy config' >&2
  exit 1
fi

systemctl reload nginx.service

status_code="$(curl -sS -o /tmp/sudharma-nginx-status.json -w '%{http_code}' "http://${PRIVATE_IP}:29100/v1/status" || true)"
if [[ "$status_code" != '200' ]]; then
  restore_config
  nginx -t >/dev/null
  systemctl reload nginx.service
  echo "wallet status route failed after nginx reload: HTTP $status_code; restored previous config" >&2
  exit 1
fi

explorer_headers="$(mktemp /tmp/sudharma-explorer-headers.XXXXXX)"
explorer_body="$(mktemp /tmp/sudharma-explorer-body.XXXXXX)"
trap 'rm -f "$backup" "$work" "$explorer_headers" "$explorer_body"' EXIT INT TERM
explorer_code="$(curl -sS -D "$explorer_headers" -o "$explorer_body" -w '%{http_code}' "http://${PRIVATE_IP}:29100/v1/explorer/status" || true)"
content_type="$(awk 'BEGIN{IGNORECASE=1} /^Content-Type:/ {sub(/\r$/, "", $0); sub(/^[^:]*:[[:space:]]*/, "", $0); print; exit}' "$explorer_headers")"

# Before the explorer-capable node binary is installed, a JSON 404 from the
# node is expected. What must no longer happen is nginx's HTML catch-all 404.
if [[ "$content_type" != application/json* ]]; then
  restore_config
  nginx -t >/dev/null
  systemctl reload nginx.service
  echo "explorer route did not reach node JSON API (HTTP $explorer_code, content-type '$content_type'); restored previous config" >&2
  exit 1
fi

case "$explorer_code" in
  200|404) ;;
  *)
    restore_config
    nginx -t >/dev/null
    systemctl reload nginx.service
    echo "unexpected explorer bridge HTTP status $explorer_code; restored previous config" >&2
    exit 1
    ;;
esac

echo "Explorer nginx allowlist ready on ${PRIVATE_IP}:29100 (explorer HTTP ${explorer_code})."
