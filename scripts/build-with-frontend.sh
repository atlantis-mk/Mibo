#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SERVER_DIR="$ROOT_DIR/mibo-server"
FRONTEND_DIST="$ROOT_DIR/dist"
EMBED_DIR="$SERVER_DIR/internal/webui/dist"
OUTPUT_PATH="${MIBO_OUTPUT:-$ROOT_DIR/build/mibo-server}"

if [ ! -f "$SERVER_DIR/go.mod" ]; then
	printf 'Initializing backend submodule...\n'
	(cd "$ROOT_DIR" && git submodule update --init --recursive -- mibo-server)
fi

printf 'Building frontend assets...\n'
rm -rf "$FRONTEND_DIST"
(cd "$ROOT_DIR" && VITE_API_BASE_URL= pnpm exec vite build --outDir "$FRONTEND_DIST")

printf 'Embedding frontend assets into backend...\n'
rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -R "$FRONTEND_DIST/." "$EMBED_DIR/"

if [ ! -f "$EMBED_DIR/index.html" ]; then
	printf 'Frontend build did not produce %s\n' "$EMBED_DIR/index.html" >&2
	exit 1
fi

printf 'Building backend binary...\n'
mkdir -p "$(dirname -- "$OUTPUT_PATH")"
(cd "$SERVER_DIR" && go build -o "$OUTPUT_PATH" ./cmd/mibo-media-server)

printf 'Built %s\n' "$OUTPUT_PATH"
