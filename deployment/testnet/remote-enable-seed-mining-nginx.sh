#!/usr/bin/env bash
# Enables /v1/mining/work and /v1/mining/submit on the seed nginx listener
# that fronts sudharma-rpcd (typically port 29100).
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-enable-seed-mining-nginx must run as root" >&2
  exit 2
fi

discover_config_files() {
  local -a files=()
  local path seen=""
  add_file() {
    local candidate="$1"
    [ -n "$candidate" ] || return 0
    [ -f "$candidate" ] || return 0
    case "$seen" in
      *"|${candidate}|"*) return 0 ;;
    esac
    seen="${seen}|${candidate}|"
    files+=("$candidate")
  }

  while IFS= read -r path; do
    add_file "$path"
  done < <(
    grep -RIl --include='*.conf' --include='*.inc' '29100' /etc/nginx /etc/sudharma /usr/local/etc/nginx 2>/dev/null | sort -u
  )

  if command -v nginx >/dev/null 2>&1; then
    while IFS= read -r path; do
      [ -n "$path" ] || continue
      if grep -q '29100' "$path" 2>/dev/null; then
        add_file "$path"
      fi
    done < <(nginx -T 2>/dev/null | awk -F: '/^# configuration file / {print $2}' | sed 's/:$//' | sort -u)
  fi

  while IFS= read -r path; do
    add_file "$path"
  done < <(
    grep -RIl --include='*.conf' --include='*.inc' -E '/v1/status|/v1/explorer/' /etc/nginx /etc/sudharma /usr/local/etc/nginx 2>/dev/null | sort -u
  )

  printf '%s\n' "${files[@]}"
}

mapfile -t config_files < <(discover_config_files)

if [ "${#config_files[@]}" -eq 0 ]; then
  echo "Could not locate sudharma public RPC nginx config" >&2
  if command -v nginx >/dev/null 2>&1; then
    nginx -T 2>/dev/null | grep -E 'listen|/v1/status|/v1/explorer|29100' | head -80 >&2 || true
  fi
  exit 1
fi

export SEED_PRIVATE_IP="${SEED_PRIVATE_IP:-}"

python3 - <<'PY' "${config_files[@]}"
import json
import os
import pathlib
import re
import sys

configs = [pathlib.Path(p) for p in sys.argv[1:] if p]
seed_private_ip = os.environ.get("SEED_PRIVATE_IP", "").strip()

def normalize_upstream_base(url: str) -> str:
    match = re.match(r"(https?://[^/]+)", url.strip())
    return match.group(1) if match else url.rstrip("/")

def iter_server_blocks(text: str):
    for match in re.finditer(r"\bserver\s*\{", text):
        start = match.start()
        brace = text.find("{", start)
        if brace == -1:
            continue
        depth = 0
        for i in range(brace, len(text)):
            if text[i] == "{":
                depth += 1
            elif text[i] == "}":
                depth -= 1
                if depth == 0:
                    yield start, i + 1, text[start:i + 1]
                    break

def server_listens_on_29100(block: str) -> bool:
    if not re.search(r"listen\s+[^;]*\b29100\b", block):
        return False
    if seed_private_ip and seed_private_ip not in block:
        return False
    return True

def server_mining_ready(block: str) -> bool:
    return (
        re.search(r"location\s+=\s+/v1/mining/work\s*\{", block) is not None
        and re.search(r"location\s+=\s+/v1/mining/submit\s*\{", block) is not None
    )

def infer_upstream_base(block: str):
    patterns = (
        r"location\s+=\s+/v1/status\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+/v1/explorer/\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+~\s+\^/v1/\([^\)]*\)[^{]*\{[^}]*?proxy_pass\s+([^;]+);",
        r"proxy_pass\s+(https?://127\.0\.0\.1:[0-9]+)",
    )
    for pattern in patterns:
        match = re.search(pattern, block, re.S)
        if match:
            return normalize_upstream_base(match.group(1))
    return None

def mining_locations_snippet(base: str) -> str:
    return f"""
    location = /v1/mining/work {{
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
        proxy_pass {base}/v1/mining/work;
    }}

    location = /v1/mining/submit {{
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 1m;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
        proxy_pass {base}/v1/mining/submit;
    }}
"""

def patch_regex_allowlist(text: str) -> tuple[str, bool]:
    pattern = re.compile(
        r"(location\s+~\s+\^/v1/\()([^)]+)(\)[^{]*\{[^}]*?proxy_pass\s+[^;]+;[^}]*\})",
        re.S,
    )
    match = pattern.search(text)
    if not match:
        return text, False
    group = match.group(2)
    if "mining/work" in group and "mining/submit" in group:
        return text, False
    updated_group = group.rstrip("|") + "|mining/work|mining/submit|"
    return text[:match.start(2)] + updated_group + text[match.end(2):], True

def insert_after_status_location(block: str, snippet: str) -> tuple[str, bool]:
    match = re.search(r"location\s+=\s+/v1/status\s*\{", block)
    if not match:
        return block, False
    brace = block.find("{", match.start())
    if brace == -1:
        return block, False
    depth = 0
    for i in range(brace, len(block)):
        if block[i] == "{":
            depth += 1
        elif block[i] == "}":
            depth -= 1
            if depth == 0:
                return block[: i + 1] + snippet + block[i + 1 :], True
    return block, False

def any_29100_server_ready(text: str) -> bool:
    for _, _, block in iter_server_blocks(text):
        if server_listens_on_29100(block) and server_mining_ready(block):
            return True
    return False

changed = []
for path in configs:
    if not path.is_file():
        continue
    text = path.read_text(encoding="utf-8")
    updated_text = text
    file_changed = False

    for start, end, block in iter_server_blocks(text):
        if not server_listens_on_29100(block):
            continue
        if server_mining_ready(block):
            continue
        base = infer_upstream_base(block)
        if not base:
            continue
        patched_block, ok = insert_after_status_location(block, mining_locations_snippet(base))
        if not ok:
            continue
        updated_text = updated_text[:start] + patched_block + updated_text[end:]
        file_changed = True
        break

    if not file_changed:
        patched, ok = patch_regex_allowlist(updated_text)
        if ok:
            updated_text = patched
            file_changed = True

    if file_changed and updated_text != text:
        path.write_text(updated_text, encoding="utf-8")
        changed.append(str(path))

if not changed:
    if any(any_29100_server_ready(path.read_text(encoding="utf-8")) for path in configs if path.is_file()):
        print(json.dumps({"seed_mining_nginx": "already_enabled"}))
    else:
        print(json.dumps({
            "seed_mining_nginx": "patch_failed",
            "config_files": [str(p) for p in configs if p.is_file()],
        }), file=sys.stderr)
        sys.exit(1)
else:
    print(json.dumps({"seed_mining_nginx": "updated", "files": changed}))
PY

nginx -t
systemctl reload nginx

nginx_smoke_url="${SEED_RPC_SMOKE_URL:-}"
if [ -z "$nginx_smoke_url" ] && [ -n "${SEED_PRIVATE_IP:-}" ]; then
  nginx_smoke_url="http://${SEED_PRIVATE_IP}:29100"
fi
if [ -z "$nginx_smoke_url" ]; then
  echo "SEED_RPC_SMOKE_URL or SEED_PRIVATE_IP is required for nginx mining smoke" >&2
  exit 1
fi
nginx_smoke_url="${nginx_smoke_url%/}"

for attempt in $(seq 1 12); do
  status_code="$(curl -sS -o /tmp/seed-mining-work-smoke.json -w '%{http_code}' \
    -X POST "${nginx_smoke_url}/v1/mining/work" \
    -H 'content-type: application/json' \
    --data '{"address":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}')"
  if [ "$status_code" = '200' ] && jq -e '.algorithm and (.block or (.header_prefix and .target))' /tmp/seed-mining-work-smoke.json >/dev/null 2>&1; then
    jq -nc --arg url "$nginx_smoke_url" '{seed_mining_nginx:"ok",mining_work:"ready",smoke_url:$url}'
    exit 0
  fi
  echo "nginx mining smoke attempt $attempt: HTTP $status_code" >&2
  head -c 400 /tmp/seed-mining-work-smoke.json 2>/dev/null >&2 || true
  echo >&2
  sleep 2
done

echo "seed nginx on ${nginx_smoke_url} did not serve POST /v1/mining/work" >&2
if command -v nginx >/dev/null 2>&1; then
  nginx -T 2>/dev/null | grep -E 'listen|/v1/mining|/v1/status|29100' | head -80 >&2 || true
fi
exit 1
