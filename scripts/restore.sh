#!/usr/bin/env bash
#
# XiaoWork Nav — restore the data directory from a snapshot.
#
#   ./scripts/restore.sh /path/to/backup-2.0.0-20261025-101500.tar.gz
#   ./scripts/restore.sh <snapshot> --compose compose.npm.yaml --yes
#
# The current data is snapshotted first, so a mistaken restore can itself be
# undone. The application is stopped for the duration.

set -Eeuo pipefail

SCRIPT_NAME="restore"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_FILE="compose.yaml"
SNAPSHOT=""
ASSUME_YES=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --yes|-y)  ASSUME_YES=1; shift ;;
    -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
    -*)        echo "unknown argument: $1" >&2; exit 2 ;;
    *)         SNAPSHOT="$1"; shift ;;
  esac
done

[[ -n "${SNAPSHOT}" ]] || { sed -n '2,10p' "$0"; exit 2; }

# shellcheck source=scripts/lib.sh
source "${SCRIPT_DIR}/lib.sh"

require_docker
require_env_file

[[ -f "${SNAPSHOT}" ]] || die "snapshot not found: ${SNAPSHOT}"
SNAPSHOT="$(cd "$(dirname "${SNAPSHOT}")" && pwd)/$(basename "${SNAPSHOT}")"
# shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
tar ${TAR_EXTRA} -tzf "${SNAPSHOT}" >/dev/null 2>&1 || die "snapshot is not a readable tar.gz archive: ${SNAPSHOT}"

DATA_DIR="$(data_host_dir)"
mkdir -p "${DATA_DIR}"

log "snapshot to restore: ${SNAPSHOT}"
log "target data directory: ${DATA_DIR}"
if [[ "${ASSUME_YES}" -ne 1 ]]; then
  printf 'This will REPLACE the current data. Continue? [y/N] '
  read -r answer
  [[ "${answer}" == "y" || "${answer}" == "Y" ]] || die "aborted by user"
fi

log "stopping the application"
compose stop "${SERVICE_NAME}" || true

SAFETY="$(snapshot_data_dir)" && log "current data saved to ${SAFETY}"

log "restoring"
rm -rf "${DATA_DIR:?}"/* 2>/dev/null || true
# shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
tar ${TAR_EXTRA} -xzf "${SNAPSHOT}" -C "${DATA_DIR}" || die "restore failed; current data is in ${SAFETY}"

log "starting the application"
compose up -d
if ! status="$(wait_for_health 120)"; then
  compose logs --tail=60 "${SERVICE_NAME}" || true
  die "the application did not become healthy after the restore (status: ${status})"
fi

log "restore complete (status: ${status})"
log "previous data kept at: ${SAFETY}"
