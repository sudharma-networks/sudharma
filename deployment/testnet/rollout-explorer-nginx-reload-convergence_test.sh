#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT_DIR/deployment/testnet/rollout-explorer-nginx-allowlist.sh"
[[ -f "$SCRIPT" ]] || { echo "missing rollout script: $SCRIPT" >&2; exit 1; }

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
FAKE_BIN="$TMP_DIR/bin"
mkdir -p "$FAKE_BIN"
CONFIG="$TMP_DIR/sudharma-wallet-proxy"
LOG="$TMP_DIR/actions.log"
COUNT_FILE="$TMP_DIR/explorer-count"
printf '0\n' > "$COUNT_FILE"
: > "$LOG"

cat > "$CONFIG" <<'EOF'
server {
    listen 172.31.10.171:29100;
    server_name _;

    location = /v1/status {
        limit_except GET { deny all; }
        proxy_pass http://127.0.0.1:28545/v1/status;
    }

    location / {
        return 404;
    }
}
EOF

cat > "$FAKE_BIN/nginx" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "nginx $*" >> "$FAKE_LOG"
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
    -D|--dump-header) headers="$2"; shift 2 ;;
    -o|--output) output="$2"; shift 2 ;;
    -w|--write-out) write_out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  */v1/status)
    [[ -z "$output" ]] || printf '{"network":"sudharma"}\n' > "$output"
    [[ -z "$write_out" ]] || printf '200'
    ;;
  */v1/explorer/status)
    count="$(cat "$EXPLORER_COUNT_FILE")"
    count=$((count + 1))
    printf '%s\n' "$count" > "$EXPLORER_COUNT_FILE"
    if (( count < 4 )); then
      [[ -z "$headers" ]] || printf 'Content-Type: text/html\r\n\r\n' > "$headers"
      [[ -z "$output" ]] || printf '<html>old worker 404</html>\n' > "$output"
      [[ -z "$write_out" ]] || printf '404'
    else
      [[ -z "$headers" ]] || printf 'Content-Type: application/json; charset=utf-8\r\n\r\n' > "$headers"
      [[ -z "$output" ]] || printf '{"error":"endpoint not found"}\n' > "$output"
      [[ -z "$write_out" ]] || printf '404'
    fi
    ;;
  *) exit 22 ;;
esac
EOF
chmod +x "$FAKE_BIN/curl"

if ! PATH="$FAKE_BIN:$PATH" \
  FAKE_LOG="$LOG" \
  EXPLORER_COUNT_FILE="$COUNT_FILE" \
  CONFIG_PATH="$CONFIG" \
  PRIVATE_IP="172.31.10.171" \
  "$SCRIPT"; then
  echo 'rollout must tolerate bounded old-worker responses after nginx reload' >&2
  exit 1
fi

count="$(cat "$COUNT_FILE")"
if (( count < 4 )); then
  echo "expected explorer probe retries during reload convergence; got $count request(s)" >&2
  exit 1
fi

grep -Fq 'systemctl reload nginx.service' "$LOG"
echo 'PASS: nginx rollout waits for explorer route convergence after reload'
