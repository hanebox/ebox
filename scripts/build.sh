#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="${COMMIT:-$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo none)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

echo "==> Building web bundle"
cd "$ROOT/web"
if [ ! -d node_modules ]; then
  bun install --frozen-lockfile
fi
bun run build

echo "==> Embedding web assets into backend"
rm -rf "$ROOT/backend/web/dist"
mkdir -p "$ROOT/backend/web/dist"
cp -R "$ROOT/web/dist/." "$ROOT/backend/web/dist/"

echo "==> Building ebox binary ($VERSION)"
mkdir -p "$ROOT/dist"
cd "$ROOT/backend"
CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags "-s -w -X ebox/app/support/version.Version=$VERSION -X ebox/app/support/version.Commit=$COMMIT -X ebox/app/support/version.Date=$DATE" \
  -o "$ROOT/dist/ebox" \
  .

echo "==> Done: $ROOT/dist/ebox"
