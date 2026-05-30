#!/usr/bin/env bash
# docker-ensure.sh — ensure a Docker container is running.
# Creates it if missing, starts it if stopped, skips if already running.
# Usage: docker-ensure.sh <name> <image> <host_port:container_port> [extra_port ...]

set -euo pipefail

NAME="$1"
IMAGE="$2"
shift 2
# remaining args are port mappings

if [ "$(docker inspect -f '{{.State.Running}}' "$NAME" 2>/dev/null)" = "true" ]; then
  echo "docker-ensure: $NAME already running"
  exit 0
fi

if docker inspect "$NAME" >/dev/null 2>&1; then
  echo "docker-ensure: $NAME exists but stopped — starting"
  docker start "$NAME"
else
  echo "docker-ensure: $NAME does not exist — creating"
  PORTS=()
  for p in "$@"; do
    PORTS+=(-p "$p")
  done
  docker run -d --name "$NAME" "${PORTS[@]}" "$IMAGE"
fi
