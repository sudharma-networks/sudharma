#!/usr/bin/env bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "error: run this installer as root" >&2
  exit 1
fi

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
bundle_root="$(CDPATH= cd -- "$script_dir/.." && pwd)"
binary="$bundle_root/sudharma-gpupow-staging"
checksums="$bundle_root/SHA256SUMS.txt"
service_src="$script_dir/sudharma-gpupow-staging.service"
nginx_example="$script_dir/nginx-staging.example.conf"

for path in "$binary" "$checksums" "$service_src" "$nginx_example"; do
  if [ ! -f "$path" ]; then
    echo "error: required staging bundle file missing: $path" >&2
    exit 1
  fi
done

(
  cd "$bundle_root"
  sha256sum -c SHA256SUMS.txt
)

if ! id -u sudharma-staging >/dev/null 2>&1; then
  useradd --system --home-dir /nonexistent --no-create-home --shell /usr/sbin/nologin sudharma-staging
fi

install -o root -g root -m 0755 "$binary" /usr/local/bin/sudharma-gpupow-staging
install -o root -g root -m 0644 "$service_src" /etc/systemd/system/sudharma-gpupow-staging.service

systemctl daemon-reload
systemctl enable --now sudharma-gpupow-staging.service

if ! systemctl is-active --quiet sudharma-gpupow-staging.service; then
  echo "error: sudharma-gpupow-staging.service did not become active" >&2
  systemctl --no-pager --full status sudharma-gpupow-staging.service || true
  exit 1
fi

echo "staging-verifier=installed"
echo "service=sudharma-gpupow-staging.service"
echo "listen=127.0.0.1:28646"
echo "nginx_example=$nginx_example"
echo "consensus_activation=disabled"
echo "block_creation=none"
echo "note=configure a separate TLS reverse proxy from nginx-staging.example.conf before physical GPU submission"
