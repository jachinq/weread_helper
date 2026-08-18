#!/bin/sh
set -eu
mkdir -p /data
if [ "$(id -u)" = "0" ]; then
  chown -R app:app /data
  exec su-exec app /app/weread-helper
fi
exec /app/weread-helper
