#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
WEB_DIR="$ROOT_DIR/web"
SERVER_DIR="$ROOT_DIR/mibo-media-server"
EMBED_DIR="$SERVER_DIR/internal/webui/dist"
OUTPUT_PATH="${MIBO_OUTPUT:-$SERVER_DIR/bin/mibo-media-server}"

(cd "$WEB_DIR" && VITE_API_BASE_URL= pnpm exec vite build --outDir dist-static)

rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -R "$WEB_DIR/dist-static/." "$EMBED_DIR/"

mkdir -p "$(dirname -- "$OUTPUT_PATH")"
(cd "$SERVER_DIR" && go build -o "$OUTPUT_PATH" ./cmd/mibo-media-server)

printf 'Built %s\n' "$OUTPUT_PATH"
