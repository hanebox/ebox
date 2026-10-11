#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Applying migrations"
(cd "$ROOT/backend" && go run . artisan migrate)

echo "==> Starting backend on http://127.0.0.1:8000"
(cd "$ROOT/backend" && go run .) &
BACKEND_PID=$!

cleanup() {
  kill "$BACKEND_PID" 2>/dev/null || true
  wait "$BACKEND_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> Starting web dev server on http://127.0.0.1:5173"
cd "$ROOT/web"
bun run dev
