#!/bin/sh
set -eu

if [ -z "${DATA_PATH:-}" ]; then
	echo "DATA_PATH is required" >&2
	exit 1
fi

dir=$(dirname "$DATA_PATH")
mkdir -p "$dir"
chown -R aero:aero "$dir"
export HOME=/home/aero
exec su-exec aero aero "$@"
