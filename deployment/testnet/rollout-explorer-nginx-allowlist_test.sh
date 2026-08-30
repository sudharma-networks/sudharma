#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT_DIR/deployment/testnet/rollout-explorer-nginx-allowlist.sh"

if [[ ! -f "$SCRIPT" ]]; then
  echo "expected production script $SCRIPT" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
FAKE_BIN="$TMP_DIR/bin"
mkdir -p "$FAKE_BIN"

cat > "$FAKE_BIN/nginx" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "nginx $*" >> "$FAKE_LOG"
if [[ "${FAKE_NGINX_TEST_FAIL:-0}" == "1" && " $* " == *" -t "* ]]; then
  exit 1
fi
exit 0
EOF
chmod +x "$FAKE_BIN/nginx"

cat > "$FAKE_BIN/systemctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "systemctl $*" >> "$FAKE_LOG"
exit 0
EOF
chmod +x "$FAKE_BIN/systemctl"

cat > "$FAKE_BIN/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
headers=''
output=''
write_out=''
url=''
while (($#)); do
  case "$1" in
    -D|--dump-header)
      headers="$2"
      shift 2
      ;;
    -o|--output)
      output="$2"
      shift 2
      ;;
    -w|--write-out)
      write_out="$2"
      shift 2
      ;;
    -*)
      shift
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done

echo "curl $url" >> "$FAKE_LOG"
case "$url" in
  */v1/status)
    [[ -z "$headers" ]] || printf 'Content-Type: application/json; charset=utf-8\r\n\r\n' > "$headers"
    [[ -z "$output" ]] || printf '{"network":"sudharma"}\n' > "$output"
    [[ -n "$output" ]] || printf '{"network":"sudharma"}\n'
    [[ -z "$write_out" ]] || printf '200'
    ;;
  */v1/explorer/status)
    [[ -z "$headers" ]] || printf 'Content-Type: application/json; charset=utf-8\r\n\r\n' > "$headers"
    [[ -z "$output" ]] || printf '{"error":"endpoint not found"}\n' > "$output"
    [[ -n "$output" ]] || printf '{"error":"endpoint not found"}\n'
    [[ -z "$write_out" ]] || printf '404'
    ;;
  *)
    exit 22
    ;;
esac
EOF
chmod +x "$FAKE_BIN/curl"

write_base_config() {
  local path="$1"
  cat > "$path" <<'EOF'
server {
    listen 172.31.10.171:29100;
    server_name _;

    add_header Cache-Control "no-store" always;

    location = /health {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545/health;
    }

    location = /ready {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545/ready;
    }

    location = /v1/status {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545/v1/status;
    }

    location ~ ^/v1/accounts/[^/]+$ {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545;
    }

    location = /v1/transactions {
        limit_except POST { deny all; }
        client_max_body_size 1m;
        proxy_pass http://127.0.0.1:28545/v1/transactions;
    }

    location ~ ^/v1/transactions/[^/]+$ {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545;
    }

    location / {
        return 404;
    }
}
EOF
}

assert_contains_once() {
  local needle="$1"
  local file="$2"
  local count
  count="$(grep -Fxc "$needle" "$file" || true)"
  [[ "$count" == "1" ]] || {
    echo "expected exactly one occurrence of: $needle; got $count" >&2
    exit 1
  }
}

CONFIG="$TMP_DIR/sudharma-wallet-proxy"
LOG="$TMP_DIR/actions.log"
: > "$LOG"
write_base_config "$CONFIG"
ORIGINAL="$TMP_DIR/original"
cp "$CONFIG" "$ORIGINAL"

PATH="$FAKE_BIN:$PATH" \
FAKE_LOG="$LOG" \
CONFIG_PATH="$CONFIG" \
PRIVATE_IP="172.31.10.171" \
"$SCRIPT"

for existing in \
  'location = /health {' \
  'location = /ready {' \
  'location = /v1/status {' \
  'location ~ ^/v1/accounts/[^/]+$ {' \
  'location = /v1/transactions {' \
  'location ~ ^/v1/transactions/[^/]+$ {'; do
  grep -Fq "$existing" "$CONFIG"
done

for explorer in \
  'location = /v1/explorer/status {' \
  'location = /v1/explorer/blocks {' \
  'location ~ "^/v1/explorer/blocks/(?:[0-9]+|[0-9a-f]{64})$" {' \
  'location = /v1/explorer/transactions {' \
  'location ~ "^/v1/explorer/transactions/[0-9a-f]{64}$" {' \
  'location ~ "^/v1/explorer/addresses/[0-9a-f]{40}$" {' \
  'location = /v1/explorer/search {'; do
  assert_contains_once "$explorer" "$CONFIG"
done

assert_contains_once '# BEGIN SUDHARMA EXPLORER READ-ONLY ALLOWLIST' "$CONFIG"
assert_contains_once '# END SUDHARMA EXPLORER READ-ONLY ALLOWLIST' "$CONFIG"
if grep -Eq 'location[[:space:]]+(\^~[[:space:]]+)?/v1/explorer/' "$CONFIG"; then
  echo 'generic explorer prefix route must not be allowed' >&2
  exit 1
fi

explorer_start="$(grep -n '# BEGIN SUDHARMA EXPLORER READ-ONLY ALLOWLIST' "$CONFIG" | cut -d: -f1)"
catch_all="$(grep -n 'location / {' "$CONFIG" | cut -d: -f1)"
(( explorer_start < catch_all )) || {
  echo 'explorer allowlist must appear before catch-all 404 route' >&2
  exit 1
}

grep -Fq 'nginx -t' "$LOG"
grep -Fq 'systemctl reload nginx.service' "$LOG"
grep -Fq 'curl http://172.31.10.171:29100/v1/status' "$LOG"
grep -Fq 'curl http://172.31.10.171:29100/v1/explorer/status' "$LOG"

# Re-running must be idempotent and must not duplicate the allowlist.
PATH="$FAKE_BIN:$PATH" \
FAKE_LOG="$LOG" \
CONFIG_PATH="$CONFIG" \
PRIVATE_IP="172.31.10.171" \
"$SCRIPT"
assert_contains_once '# BEGIN SUDHARMA EXPLORER READ-ONLY ALLOWLIST' "$CONFIG"

# A failed nginx validation must restore the original configuration and avoid reload.
FAIL_CONFIG="$TMP_DIR/failing-wallet-proxy"
FAIL_LOG="$TMP_DIR/failing-actions.log"
: > "$FAIL_LOG"
write_base_config "$FAIL_CONFIG"
cp "$FAIL_CONFIG" "$TMP_DIR/failing-original"
set +e
PATH="$FAKE_BIN:$PATH" \
FAKE_LOG="$FAIL_LOG" \
FAKE_NGINX_TEST_FAIL=1 \
CONFIG_PATH="$FAIL_CONFIG" \
PRIVATE_IP="172.31.10.171" \
"$SCRIPT"
status=$?
set -e
[[ "$status" -ne 0 ]] || {
  echo 'expected nginx validation failure' >&2
  exit 1
}
cmp -s "$FAIL_CONFIG" "$TMP_DIR/failing-original" || {
  echo 'nginx configuration was not restored after validation failure' >&2
  exit 1
}
if grep -Fq 'systemctl reload nginx.service' "$FAIL_LOG"; then
  echo 'nginx must not reload after validation failure' >&2
  exit 1
fi

echo 'PASS: explorer nginx allowlist rollout contract'
