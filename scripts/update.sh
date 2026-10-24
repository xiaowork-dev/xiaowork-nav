#!/usr/bin/env bash
#
# XiaoWork Nav — safe one-command update.
#
# Steps:
#   1. read current version from deploy/.env
#   2. stop the app and take a consistent snapshot of the data directory
#   3. switch APP_VERSION atomically
#   4. pull the target image and start
#   5. verify health; on failure restore the previous version and restart
#
# The data directory is snapshotted while the container is stopped so that the
# SQLite database is captured consistently. A proper online backup arrives with
# the in-app backup feature.
#
# Usage:
#   ./scripts/update.sh [VERSION] [--compose compose.npm.yaml] [--yes]
#   VERSION defaults to "latest".

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"

TARGET_VERSION=""
COMPOSE_FILE="compose.yaml"
ASSUME_YES=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --yes|-y)  ASSUME_YES=1; shift ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    -*)        echo "unknown argument: $1" >&2; exit 2 ;;
    *)         TARGET_VERSION="$1"; shift ;;
  esac
done
TARGET_VERSION="${TARGET_VERSION:-latest}"

log()  { printf '\033[1;34m[update]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

command -v docker >/dev/null 2>&1 || die "docker is required but was not found"
docker compose version >/dev/null 2>&1 || die "the Docker Compose plugin is required"
[[ -f "${ENV_FILE}" ]] || die "missing ${ENV_FILE}; run deploy.sh first"

COMPOSE_PATH="${DEPLOY_DIR}/${COMPOSE_FILE}"
[[ -f "${COMPOSE_PATH}" ]] || die "compose file not found: ${COMPOSE_PATH}"

compose() {
  docker compose -f "${COMPOSE_PATH}" --env-file "${ENV_FILE}" "$@"
}

get_env() {
  local key="$1"
  grep -E "^${key}=" "${ENV_FILE}" | tail -n1 | cut -d= -f2- || true
}

set_env() {
  local key="$1" value="$2" tmp
  tmp="$(mktemp)"
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    awk -v k="${key}" -v v="${value}" \
      'BEGIN{FS=OFS="="} $1==k{ $0=k"="v } {print}' "${ENV_FILE}" > "${tmp}"
  else
    cat "${ENV_FILE}" > "${tmp}"
    printf '%s=%s\n' "${key}" "${value}" >> "${tmp}"
  fi
  mv "${tmp}" "${ENV_FILE}"
}

CURRENT_VERSION="$(get_env APP_VERSION)"
CURRENT_VERSION="${CURRENT_VERSION:-unknown}"

if [[ "${TARGET_VERSION}" == "${CURRENT_VERSION}" ]]; then
  log "already at version ${TARGET_VERSION}; nothing to do"
  exit 0
fi

DATA_DIR="$(get_env XN_DATA_HOST_DIR)"
DATA_DIR="${DATA_DIR:-./data}"
case "${DATA_DIR}" in
  /*) : ;;                       # absolute
  *) DATA_DIR="${DEPLOY_DIR}/${DATA_DIR#./}" ;;
esac
[[ -d "${DATA_DIR}" ]] || die "data directory not found: ${DATA_DIR}"

BACKUP_DIR="${DATA_DIR}/../xiaowork-nav-backups"
mkdir -p "${BACKUP_DIR}"
STAMP="$(date -u +%Y%m%d-%H%M%S)"
SNAPSHOT="${BACKUP_DIR}/pre-update-${CURRENT_VERSION}-${STAMP}.tar.gz"

log "current version: ${CURRENT_VERSION} -> target: ${TARGET_VERSION}"
if [[ "${ASSUME_YES}" -ne 1 ]]; then
  printf 'Proceed? [y/N] '
  read -r answer
  [[ "${answer}" == "y" || "${answer}" == "Y" ]] || die "aborted by user"
fi

log "stopping app for a consistent snapshot"
compose stop app || true

log "snapshotting data directory -> ${SNAPSHOT}"
tar -czf "${SNAPSHOT}" -C "${DATA_DIR}" . || die "snapshot failed; aborting before any change"
log "snapshot size: $(du -h "${SNAPSHOT}" | cut -f1)"

rollback() {
  warn "update failed — rolling back to ${CURRENT_VERSION}"
  set_env APP_VERSION "${CURRENT_VERSION}"
  compose up -d || warn "rollback restart failed; manual attention required"
  warn "data snapshot kept at: ${SNAPSHOT}"
  warn "if the new version ran an irreversible migration, restore the snapshot before starting the old version"
}

log "switching version"
set_env APP_VERSION "${TARGET_VERSION}"

log "pulling image"
if ! compose pull; then
  rollback
  die "image pull failed"
fi

log "starting updated app"
if ! compose up -d; then
  rollback
  die "container start failed"
fi

log "verifying health"
cid="$(compose ps -q app)"
deadline=$((SECONDS + 90))
status="unknown"
while [[ ${SECONDS} -lt ${deadline} ]]; do
  status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${cid}" 2>/dev/null || echo unknown)"
  [[ "${status}" == "healthy" || "${status}" == "running" ]] && break
  [[ "${status}" == "unhealthy" ]] && break
  sleep 3
done

if [[ "${status}" != "healthy" && "${status}" != "running" ]]; then
  compose logs --tail=60 app || true
  rollback
  die "health check failed (status: ${status})"
fi

log "update complete: ${CURRENT_VERSION} -> ${TARGET_VERSION} (status: ${status})"
log "snapshot retained at: ${SNAPSHOT}"
