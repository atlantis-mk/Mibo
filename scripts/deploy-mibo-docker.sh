#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
REMOTE_HOST="${REMOTE_HOST:-root@10.0.0.4}"
CONTAINER_NAME="${CONTAINER_NAME:-mibo}"
IMAGE_TAG="${IMAGE_TAG:-mibo-webui:deploy-$(date +%Y%m%d%H%M%S)}"
HTTP_PORT="${HTTP_PORT:-18081}"
DATA_BIND="${DATA_BIND:-/opt/mibo/data}"
MEDIA_MOUNT="${MEDIA_MOUNT:-f6b2615f8a0abbc44b72ecbf45210f963b8973805fa859fc39cda31eddbaa502:/media}"

EXISTING_MOUNTS=$(ssh "$REMOTE_HOST" "docker inspect '$CONTAINER_NAME' --format '{{range .Mounts}}{{printf \"%s|%s|%s|%s\\n\" .Type .Destination .Source .Name}}{{end}}' 2>/dev/null || true")

if [ -n "$EXISTING_MOUNTS" ]; then
	while IFS='|' read -r type destination source name; do
		if [ "$destination" = "/data" ] && [ "$type" = "bind" ] && [ -n "$source" ]; then
			DATA_BIND=$source
		fi
		if [ "$destination" = "/media" ]; then
			if [ "$type" = "volume" ] && [ -n "$name" ]; then
				MEDIA_MOUNT="$name:/media"
			elif [ "$type" = "bind" ] && [ -n "$source" ]; then
				MEDIA_MOUNT="$source:/media"
			fi
		fi
	done <<EOF
$EXISTING_MOUNTS
EOF
fi

docker build --platform linux/amd64 -t "$IMAGE_TAG" "$ROOT_DIR"
docker save "$IMAGE_TAG" | ssh -o StrictHostKeyChecking=accept-new "$REMOTE_HOST" docker load

ssh "$REMOTE_HOST" "mkdir -p '$DATA_BIND' && docker rm -f '$CONTAINER_NAME' >/dev/null 2>&1 || true && docker run -d --name '$CONTAINER_NAME' --restart unless-stopped -p '$HTTP_PORT:$HTTP_PORT' -e MIBO_HTTP_ADDR=':$HTTP_PORT' -e MIBO_DATABASE_DSN='/data/mibo.db' -v '$DATA_BIND:/data' -v '$MEDIA_MOUNT' '$IMAGE_TAG'"

ssh "$REMOTE_HOST" "docker ps --filter name='^/$CONTAINER_NAME$' --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'"
