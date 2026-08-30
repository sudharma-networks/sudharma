#!/usr/bin/env bash
# Runs on the testnet seed host (via SSM or operator SSH).
# Ensures the demand miner supervisor is installed, configured, and running.
set -euo pipefail

repo_ref="${SUDHARMA_REPO_REF:-feature/faucet-recovery-stage2}"
repo_url="${SUDHARMA_REPO_URL:-https://github.com/sudharma-networks/sudharma.git}"
workdir="${SUDHARMA_DEMAND_MINER_SRC:-/var/lib/sudharma-demand-miner/src}"
config_example="${SUDHARMA_DEMAND_MINER_CONFIG:-deployment/testnet/demand-miner.seed1-live.example.json}"

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "remote-ensure-demand-miner must run as root on the seed host" >&2
    exit 2
  fi
}

ensure_user() {
  if ! id -u sudharma-miner >/dev/null 2>&1; then
    useradd --system --home /nonexistent --shell /usr/sbin/nologin sudharma-miner
  fi
}

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi
  if command -v apt-get >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq golang-go git curl ca-certificates
    command -v go >/dev/null 2>&1
    return 0
  fi
  echo "go toolchain is required on the seed host to build demand miner binaries" >&2
  exit 2
}

sync_repo() {
  mkdir -p "$(dirname "$workdir")"
  if [ ! -d "$workdir/.git" ]; then
    git clone --depth 1 --branch "$repo_ref" "$repo_url" "$workdir"
  else
    git -C "$workdir" fetch origin "$repo_ref"
    git -C "$workdir" checkout "$repo_ref"
    git -C "$workdir" pull --ff-only origin "$repo_ref"
  fi
}

build_and_install() {
  local build_dir
  build_dir="$(mktemp -d /var/lib/sudharma-demand-miner/build-XXXXXX)"
  trap 'rm -rf "$build_dir"' RETURN

  export HOME="${HOME:-/var/lib/sudharma-demand-miner}"
  export GOPATH="${GOPATH:-/var/lib/sudharma-demand-miner/go}"
  export GOMODCACHE="${GOMODCACHE:-$GOPATH/pkg/mod}"
  export GOCACHE="${GOCACHE:-$GOPATH/cache}"
  mkdir -p "$HOME" "$GOMODCACHE" "$GOCACHE" "$GOPATH"

  (
    cd "$workdir"
    go build -trimpath -o "$build_dir/sudharma-demand-miner" ./cmd/sudharma-demand-miner
    go build -trimpath -o "$build_dir/sudharmad" ./cmd/sudharmad
  )

  if [ ! -f "$workdir/$config_example" ]; then
    echo "config example missing: $workdir/$config_example" >&2
    exit 2
  fi

  install -d -m 0755 /etc/sudharma
  if [ ! -f /etc/sudharma/demand-miner.json ]; then
    install -m 0640 "$workdir/$config_example" /etc/sudharma/demand-miner.json
    chown root:sudharma-miner /etc/sudharma/demand-miner.json
  fi

  DEMAND_MINER_BIN="$build_dir/sudharma-demand-miner" \
  SUDHARMAD_BIN="$build_dir/sudharmad" \
  bash "$workdir/deployment/testnet/install-demand-miner.sh" --enable

  systemctl is-active --quiet sudharma-demand-miner.service
  curl --fail --silent --max-time 5 http://127.0.0.1:28545/v1/status | tee /tmp/demand-miner-status-after-ensure.json
}

main() {
  require_root
  ensure_user
  ensure_go
  sync_repo
  build_and_install
  printf '{"demand_miner_ensure":"ok","service":"active"}\n'
}

main "$@"
