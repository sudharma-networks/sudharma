#!/usr/bin/env bash
# Enables /v1/mining/work and /v1/mining/submit on the seed nginx listener
# that fronts sudharma-rpcd (typically port 29100).
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "remote-enable-seed-mining-nginx must run as root" >&2
  exit 2
fi

mapfile -t config_files < <(
  grep -RIl --include='*.conf' --include='*.inc' '29100' /etc/nginx 2>/dev/null | sort -u
)

if [ "${#config_files[@]}" -eq 0 ]; then
  echo "No nginx config listening on 29100 was found under /etc/nginx" >&2
  exit 1
fi

python3 - <<'PY' "${config_files[@]}"
import pathlib
import re
import sys

configs = [pathlib.Path(p) for p in sys.argv[1:]]
snippet = """
    location = /v1/mining/work {
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
        proxy_pass UPSTREAM;
    }

    location = /v1/mining/submit {
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 1m;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
        proxy_pass UPSTREAM;
    }
"""

changed = []
for path in configs:
    text = path.read_text(encoding="utf-8")
    if "/v1/mining/work" in text and "/v1/mining/submit" in text:
        continue
    upstream = None
    for pattern in (
        r"location\s+=\s+/v1/status\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+/v1/explorer/\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+/v1/status\s*\{[^}]*?proxy_pass\s+([^;]+);",
        r"location\s+~\s+\^/v1/\([^\)]*status[^\)]*\)[^{]*\{[^}]*?proxy_pass\s+([^;]+);",
    ):
        match = re.search(pattern, text, re.S)
        if match:
            upstream = match.group(1).strip()
            break
    if not upstream:
        raise SystemExit(f"Could not infer sudharma upstream proxy_pass in {path}")
    insert = snippet.replace("UPSTREAM", upstream)
    marker = "    # sudharma gpu mining routes"
    if marker in text:
        continue
    server_idx = text.find("listen 29100")
    if server_idx == -1:
        server_idx = text.find("listen\t29100")
    if server_idx == -1:
        raise SystemExit(f"Could not locate listen 29100 server block in {path}")
    brace = text.find("{", server_idx)
    if brace == -1:
        raise SystemExit(f"Malformed nginx server block in {path}")
    depth = 0
    end = None
    for i in range(brace, len(text)):
        ch = text[i]
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                end = i
                break
    if end is None:
        raise SystemExit(f"Could not close server block in {path}")
    updated = text[:end] + marker + insert + text[end:]
    path.write_text(updated, encoding="utf-8")
    changed.append(str(path))

if not changed:
    print('{"seed_mining_nginx":"already_enabled"}')
else:
    import json
    print(json.dumps({"seed_mining_nginx": "updated", "files": changed}))
PY

nginx -t
systemctl reload nginx

for attempt in $(seq 1 20); do
  if curl -fsS -X POST "http://127.0.0.1:29100/v1/mining/work" \
    -H 'content-type: application/json' \
    --data '{"address":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}' \
    | tee /tmp/seed-mining-work-smoke.json \
    | jq -e '.algorithm and .block' >/dev/null 2>&1; then
    jq -nc '{seed_mining_nginx:"ok",mining_work:"ready"}'
    exit 0
  fi
  sleep 3
done

echo "nginx updated but loopback mining work smoke check did not succeed" >&2
cat /tmp/seed-mining-work-smoke.json 2>/dev/null || true
exit 1
