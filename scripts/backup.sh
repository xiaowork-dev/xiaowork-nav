#!/usr/bin/env bash
#
# XiaoWork Nav — back up the data directory (database + uploaded icons).
#
#   ./scripts/backup.sh                     # stop, snapshot, start
#   ./scripts/backup.sh --keep 10           # also prune older snapshots
#   ./scripts/backup.sh --compose compose.npm.yaml
#   ./scripts/backup.sh --out /mnt/backups
#
# The application is stopped for a few seconds so the SQLite database is
# captured consistently (a live copy of a WAL database can be corrupt).

set -Eeuo pipefail

SCRIPT_NAME="backup"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_FILE="compose.yaml"
KEEP=0
OUT_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --keep)    KEEP="${2:?missing value for --keep}"; shift 2 ;;
    --out)     OUT_DIR="${2:?missing value for --out}"; shift 2 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

# shellcheck source=scripts/lib.sh
source "${SCRIPT_DIR}/lib.sh"

require_docker
require_env_file

DATA_DIR="$(data_host_dir)"
[[ -d "${DATA_DIR}" ]] || die "data directory not found: ${DATA_DIR}"
BACKUP_DIR="${OUT_DIR:-$(dirname "${DATA_DIR}")/xiaowork-nav-backups}"
mkdir -p "${BACKUP_DIR}"

VERSION="$(get_env APP_VERSION)"
STAMP="$(date -u +%Y%m%d-%H%M%S)"
TARGET="${BACKUP_DIR}/backup-${VERSION:-unknown}-${STAMP}.tar.gz"

was_running=0
if [[ -n "$(container_id)" ]]; then
  was_running=1
  log "stopping the application for a consistent snapshot"
  compose stop "${SERVICE_NAME}" || true
fi

log "snapshotting ${DATA_DIR}"
# shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
tar ${TAR_EXTRA} -czf "${TARGET}" -C "${DATA_DIR}" . || {
  if [[ "${was_running}" -eq 1 ]]; then compose up -d || true; fi
  die "snapshot failed"
}
log "wrote ${TARGET} ($(du -h "${TARGET}" 2>/dev/null | cut -f1 || echo '?') )"

if [[ "${was_running}" -eq 1 ]]; then
  log "starting the application again"
  compose up -d
  if ! status="$(wait_for_health 120)"; then
    warn "the application did not become healthy after the backup (status: ${status})"
    compose logs --tail=40 "${SERVICE_NAME}" || true
  else
    log "application healthy (${status})"
  fi
fi

if [[ "${KEEP}" -gt 0 ]]; then
  log "keeping the newest ${KEEP} snapshots"
  # shellcheck disable=SC2012
  ls -1t "${BACKUP_DIR}"/backup-*.tar.gz 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
    log "pruning $(basename "${old}")"
    rm -f "${old}"
  done
fi

printf '\nBackup complete: %s\n' "${TARGET}"
