#!/bin/sh
set -eu
mkdir -p "$DATA_DIR"
chown 10001:10001 "$DATA_DIR"
exec su-exec 10001:10001 "$@"
