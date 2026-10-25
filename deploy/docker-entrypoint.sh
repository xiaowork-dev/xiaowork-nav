#!/bin/sh
#
# XiaoWork Nav container entrypoint.
#
# The container runs as an unprivileged user, but /data is a bind mount whose
# owner is decided by the host. This script makes sure the persistent
# directories are usable *before* the application starts, and otherwise fails
# fast with an actionable message instead of a confusing "permission denied"
# much later (for example when the first icon is uploaded).
#
# It never asks for root and never loosens permissions: the fix is always to
# give the directory to the container user, or to run the container as the
# directory's owner (see XN_UID / XN_GID in deploy/.env).

set -eu

APP_BINARY="/usr/local/bin/xiaowork-nav"

DATA_DIR="${XN_DATA_DIR:-/data}"
UPLOADS_DIR="${XN_UPLOADS_DIR:-${DATA_DIR}/uploads}"
BACKUPS_DIR="${XN_BACKUPS_DIR:-${DATA_DIR}/backups}"

RUN_UID="$(id -u)"
RUN_GID="$(id -g)"

fail() {
  cat >&2 <<EOF

[error] $1

The data directory is bind-mounted from the host, so it must be writable by
uid ${RUN_UID} / gid ${RUN_GID} — the user this container runs as.

Fix it with one of these, then start the container again:

  1. Let the install script do it (recommended):
       sudo ./scripts/install.sh

  2. Give the directory to the container user (no 777, no root process):
       sudo chown -R ${RUN_UID}:${RUN_GID} <host data directory>

  3. Run the container as the directory's owner: set XN_UID and XN_GID in
     deploy/.env to that owner's ids (run: id -u / id -g) and re-run
     docker compose -f deploy/compose.yaml up -d

Never use chmod 777: it would expose your database and icons to every local
account on the host.

EOF
  exit 1
}

prepare() {
  local dir="$1"
  mkdir -p "${dir}" 2>/dev/null || fail "cannot create ${dir}"
  if ! touch "${dir}/.xn-write-test" 2>/dev/null; then
    fail "${dir} is not writable"
  fi
  rm -f "${dir}/.xn-write-test"
}

prepare "${DATA_DIR}"
prepare "${UPLOADS_DIR}"
prepare "${BACKUPS_DIR}"

# `docker run <image> --flag` must reach the application binary, while a bare
# run starts it normally (the standard entrypoint/CMD split).
if [ "$#" -eq 0 ]; then
  set -- "${APP_BINARY}"
elif [ "${1#-}" != "$1" ]; then
  set -- "${APP_BINARY}" "$@"
fi

exec "$@"
