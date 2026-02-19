#!/usr/bin/env bash
set -euo pipefail

echo "[1/3] Build web (bun)"
pushd "$(dirname "$0")/../web" >/dev/null
bun install
bun run build
popd >/dev/null

echo "[2/3] Run backend tests"
pushd "$(dirname "$0")/.." >/dev/null
go test ./...

echo "[3/3] Build backend binary with VERSION"
ver="$(tr -d '\r\n' < VERSION)"
commit="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=${ver}' -X 'github.com/QuantumNous/new-api/common.Commit=${commit}'" -o new-api .
echo "Built ./new-api version: $(./new-api --version)"

echo ""
echo "Run: ./new-api"
echo "Default SQLite path: one-api.db in current working directory (set SQLITE_PATH to change)."
popd >/dev/null
