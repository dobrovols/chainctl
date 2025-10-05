#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACT_DIR="$ROOT_DIR/artifacts/dry-run"
mkdir -p "$ARTIFACT_DIR"

run_cmd() {
  local name="$1"; shift
  local logfile="$ARTIFACT_DIR/${name}.txt"
  echo "Running: $*" >&2
  if ! (cd "$ROOT_DIR" && GO111MODULE=on GOCACHE="$ROOT_DIR/.gocache" "$@" >"$logfile" 2>&1); then
    echo "Command failed; captured output in $logfile" >&2
  fi
}

VALUES_FILE="${CHAINCTL_VALUES_FILE:-test/e2e/testdata/values.enc}"
PASS="${CHAINCTL_VALUES_PASSPHRASE:-secret}"
CLUSTER_ENDPOINT="${CHAINCTL_CLUSTER_ENDPOINT:-https://cluster.local}"
K3S_VERSION="${CHAINCTL_K3S_VERSION:-v1.30.2+k3s1}"
JOIN_TOKEN="${CHAINCTL_JOIN_TOKEN:-}" 

run_cmd cluster-install go run ./cmd/chainctl cluster install \
  --bootstrap \
  --values-file "$VALUES_FILE" \
  --values-passphrase "$PASS" \
  --dry-run

run_cmd app-upgrade go run ./cmd/chainctl app upgrade \
  --cluster-endpoint "$CLUSTER_ENDPOINT" \
  --values-file "$VALUES_FILE" \
  --values-passphrase "$PASS" \
  --output json || true

run_cmd cluster-upgrade go run ./cmd/chainctl cluster upgrade \
  --cluster-endpoint "$CLUSTER_ENDPOINT" \
  --k3s-version "$K3S_VERSION" \
  --output json || true

if [[ -n "$JOIN_TOKEN" ]]; then
  run_cmd node-join go run ./cmd/chainctl node join \
    --cluster-endpoint "$CLUSTER_ENDPOINT" \
    --role "${CHAINCTL_JOIN_ROLE:-worker}" \
    --token "$JOIN_TOKEN" \
    --output json || true
fi
