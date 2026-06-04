#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
SERVER_DIR="$ROOT_DIR/mibo-server"
BACKEND_ADDR="${MIBO_HTTP_ADDR:-:8096}"
STATUS_DIR=$(mktemp -d "${TMPDIR:-/tmp}/mibo-dev.XXXXXX")

pids=()

cleanup() {
	local status=$?
	trap - EXIT INT TERM

	for pid in "${pids[@]}"; do
		if kill -0 "$pid" 2>/dev/null; then
			pkill -TERM -P "$pid" 2>/dev/null || true
			kill -TERM "$pid" 2>/dev/null || true
		fi
	done

	for pid in "${pids[@]}"; do
		wait "$pid" 2>/dev/null || true
	done

	rm -rf "$STATUS_DIR"
	exit "$status"
}

start_process() {
	local label=$1
	local workdir=$2
	shift 2

	(
		set +e
		cd "$workdir"
		"$@" 2>&1 | sed -u "s/^/[$label] /"
		status=${PIPESTATUS[0]}
		printf '%s\n' "$status" > "$STATUS_DIR/$label.status"
		exit "$status"
	) &

	pids+=("$!")
}

trap cleanup EXIT INT TERM

if [ ! -f "$SERVER_DIR/go.mod" ]; then
	printf 'Initializing backend submodule...\n'
	(cd "$ROOT_DIR" && git submodule update --init --recursive -- mibo-server)
fi

printf 'Starting Mibo full-stack development environment...\n'
printf '  frontend: pnpm dev\n'
printf '  backend:  MIBO_HTTP_ADDR=%s go run ./cmd/mibo-media-server\n' "$BACKEND_ADDR"
printf '\nPress Ctrl+C to stop both services.\n\n'

start_process backend "$SERVER_DIR" env MIBO_HTTP_ADDR="$BACKEND_ADDR" go run ./cmd/mibo-media-server
start_process frontend "$ROOT_DIR" pnpm dev

while true; do
	for status_file in "$STATUS_DIR"/*.status; do
		[ -e "$status_file" ] || continue
		status=$(cat "$status_file")
		service=$(basename "$status_file" .status)
		printf '\n%s exited with status %s; stopping the other service.\n' "$service" "$status" >&2
		exit "$status"
	done
	sleep 1
done
