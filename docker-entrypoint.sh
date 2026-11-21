#!/bin/sh
set -eu

if [ -d /backups ]; then
	chown aero:aero /backups
fi
export HOME=/home/aero
exec su-exec aero aero "$@"
