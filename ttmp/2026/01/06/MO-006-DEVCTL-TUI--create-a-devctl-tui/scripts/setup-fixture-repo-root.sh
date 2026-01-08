#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
setup-fixture-repo-root.sh

Creates a temporary "repo-root" directory that looks like a realistic devctl target
repo, suitable for testing:
  - devctl tui dashboard/logs/exit diagnostics
  - pipeline view + validation issue navigation

The fixture includes:
  - $REPO_ROOT/.devctl.yaml using the built-in e2e plugin
  - $REPO_ROOT/bin/http-echo and $REPO_ROOT/bin/log-spewer

Usage:
  ./setup-fixture-repo-root.sh

Output:
  Prints the created REPO_ROOT path to stdout.

Notes:
  - Run from the devctl repo root (where go.work/go.mod lives).
  - Cleanup is manual: rm -rf "$REPO_ROOT"
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ ! -f "go.mod" ]]; then
  echo "error: run this from the devctl repo root (expected ./go.mod)" >&2
  exit 2
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "error: python3 is required for picking a free port" >&2
  exit 2
fi

REPO_ROOT="$(mktemp -d -t devctl-tui-fixture-XXXXXX)"

mkdir -p "$REPO_ROOT/bin"

# Build tiny fixture services into $REPO_ROOT/bin.
# We set GOWORK=off so this works even when invoked from a go.work workspace.
GOWORK=off go build -o "$REPO_ROOT/bin/http-echo" ./testapps/cmd/http-echo
GOWORK=off go build -o "$REPO_ROOT/bin/log-spewer" ./testapps/cmd/log-spewer

# Pick a free port (mac/linux compatible).
PORT="$(python3 - <<'PY'
import socket
s=socket.socket()
s.bind(("127.0.0.1",0))
print(s.getsockname()[1])
s.close()
PY
)"

PLUGIN="$(pwd)/testdata/plugins/e2e/plugin.py"

cat >"$REPO_ROOT/.devctl.yaml" <<YAML
plugins:
  - id: e2e
    path: python3
    args:
      - "$PLUGIN"
    env:
      DEVCTL_HTTP_ECHO_BIN: "$REPO_ROOT/bin/http-echo"
      DEVCTL_LOG_SPEWER_BIN: "$REPO_ROOT/bin/log-spewer"
      DEVCTL_HTTP_ECHO_PORT: "$PORT"
    priority: 10
YAML

echo "$REPO_ROOT"
