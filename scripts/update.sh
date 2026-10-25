#!/usr/bin/env bash
#
# XiaoWork Nav — safe one-command update.
#
#   ./scripts/update.sh v2.0.1              # update to a specific release
#   ./scripts/update.sh latest              # update to the latest release
#   ./scripts/update.sh v2.0.1 --compose compose.npm.yaml
#   ./scripts/update.sh v2.0.1 --local      # use a locally built image (no pull)
#   ./scripts/update.sh v2.0.1 --yes        # skip the confirmation prompt
#
# What it does:
#   1. records the current version and database schema version
#   2. stops the app and takes a consistent snapshot of the data directory
#      (database + uploaded icons + backups)
#   3. switches APP_VERSION atomically and starts the new image
#   4. waits for /healthz and verifies the schema version
#   5. on failure: restores the snapshot when the new build migrated the
#      database, then starts the previous image again
#
# Data is never deleted. See docs/updating.md for the manual equivalents.

set -Eeuo pipefail

SCRIPT_NAME="update"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_FILE="compose.yaml"
TARGET_VERSION=""
ASSUME_YES=0
LOCAL_IMAGE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --yes|-y)  ASSUME_YES=1; shift ;;
    --local)   LOCAL_IMAGE=1; shift ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    -*)        echo "unknown argument: $1" >&2; exit 2 ;;
    *)         TARGET_VERSION="$1"; shift ;;
  esac
done
TARGET_VERSION="${TARGET_VERSION:-latest}"

# shellcheck source=scripts/lib.sh
source "${SCRIPT_DIR}/lib.sh"

require_docker
require_env_file
validate_compose

CURRENT_VERSION="$(get_env APP_VERSION)"
CURRENT_VERSION="${CURRENT_VERSION:-unknown}"

if [[ "${TARGET_VERSION}" == "${CURRENT_VERSION}" ]]; then
  log "already on ${TARGET_VERSION}; nothing to do"
  exit 0
fi

log "current version: ${CURRENT_VERSION}  ->  target: ${TARGET_VERSION}"
if [[ "${ASSUME_YES}" -ne 1 ]]; then
  printf 'Proceed? [y/N] '
  read -r answer
  [[ "${answer}" == "y" || "${answer}" == "Y" ]] || die "aborted by user"
fi

# --- 1. record the current state -------------------------------------------
SCHEMA_BEFORE="$(app_schema_version || true)"
if [[ -n "${SCHEMA_BEFORE}" ]]; then
  log "current database schema version: ${SCHEMA_BEFORE}"
else
  warn "could not read the current schema version (app may be stopped)"
fi

# --- 2. stop + snapshot ----------------------------------------------------
log "stopping the application for a consistent snapshot"
compose stop "${SERVICE_NAME}" || true

SNAPSHOT="$(snapshot_data_dir)"
log "snapshot written to ${SNAPSHOT}"

rollback() {
  local reason="$1"
  warn "update failed: ${reason}"

  if [[ -n "${SCHEMA_BEFORE}" ]]; then
    local schema_after
    schema_after="$(app_schema_version || true)"
    if [[ -n "${schema_after}" && "${schema_after}" -gt "${SCHEMA_BEFORE}" ]]; then
      warn "the new build migrated the database (schema ${SCHEMA_BEFORE} -> ${schema_after});"
      warn "the previous image cannot read it, so the data snapshot must be restored first."
      restore_snapshot "${SNAPSHOT}"
    fi
  fi

  log "switching back to ${CURRENT_VERSION}"
  set_env APP_VERSION "${CURRENT_VERSION}"
  if compose up -d; then
    log "rolled back to ${CURRENT_VERSION}"
  else
    warn "rollback start failed; manual attention required"
  fi
  warn "data snapshot kept at: ${SNAPSHOT}"
  exit 1
}

restore_snapshot() {
  local snapshot="$1" data_dir
  data_dir="$(data_host_dir)"
  log "restoring data from ${snapshot}"
  rm -rf "${data_dir:?}"/* 2>/dev/null || true
  # shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
  tar ${TAR_EXTRA} -xzf "${snapshot}" -C "${data_dir}" || die "failed to restore snapshot ${snapshot}"
}

# --- 3. switch version and start ------------------------------------------
log "switching APP_VERSION to ${TARGET_VERSION}"
set_env APP_VERSION "${TARGET_VERSION}"

if [[ "${LOCAL_IMAGE}" -eq 0 ]]; then
  log "pulling $(image_ref)"
  if ! compose pull; then
    rollback "image pull failed"
  fi
fi

log "starting the updated application"
if ! compose up -d; then
  rollback "container start failed"
fi

# --- 4. verify -------------------------------------------------------------
log "verifying health"
if ! status="$(wait_for_health 120)"; then
  compose logs --tail=80 "${SERVICE_NAME}" || true
  rollback "health check failed (status: ${status})"
fi

SCHEMA_AFTER="$(app_schema_version || true)"
if [[ -n "${SCHEMA_BEFORE}" && -n "${SCHEMA_AFTER}" ]]; then
  if [[ "${SCHEMA_AFTER}" -lt "${SCHEMA_BEFORE}" ]]; then
    rollback "schema version went backwards (${SCHEMA_BEFORE} -> ${SCHEMA_AFTER})"
  fi
  log "database schema version: ${SCHEMA_BEFORE} -> ${SCHEMA_AFTER}"
fi

log "update complete: ${CURRENT_VERSION} -> ${TARGET_VERSION} (status: ${status})"
log "snapshot retained at: ${SNAPSHOT}"
log "to roll back manually: ./scripts/restore.sh ${SNAPSHOT} && APP_VERSION=${CURRENT_VERSION} ./scripts/deploy.sh --no-pull"
