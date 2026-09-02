#!/bin/sh
set -e

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

case "$PUID" in
  ''|*[!0-9]*)
    echo "PUID must be a numeric uid, got: $PUID" >&2
    exit 1
    ;;
esac
case "$PGID" in
  ''|*[!0-9]*)
    echo "PGID must be a numeric gid, got: $PGID" >&2
    exit 1
    ;;
esac

mkdir -p /config /downloads
chown -R "$PUID:$PGID" /config /downloads

exec su-exec "$PUID:$PGID" /usr/local/bin/mediator "$@"
