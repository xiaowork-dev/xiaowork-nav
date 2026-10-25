#!/usr/bin/env bash
#
# XiaoWork Nav — safe one-command update.
#
#   ./scripts/update.sh v2.0.1              # update to a specific release
#   ./scripts/update.sh latest              # update to the newest release
#   ./scripts/update.sh latest --compose compose.npm.yaml
#   ./scripts/update.sh v2.0.1 --no-pull    # use an image already on the host
#   ./scripts/update.sh v2.0.1 --yes        # skip the confirmation prompt
#
# Behaviour
#   * "latest" (and other mutable tags) always check the registry; when the
#     digest is unchanged the container is left alone instead of being
#     recreated.
#   * A pinned version that is already deployed exits immediately.
#   * Before anything changes, a consistent snapshot of the data directory is
#     taken (database + uploaded icons + in-app backups).
#   * On any failure the previous version is restored and verified, and the
#     pre-update snapshot is restored first whenever the new build migrated the
#     database (an older image cannot read a newer schema).
#
# Data is never deleted. See docs/updating.md.

set -Eeuo pipefail

SCRIPT_NAME="update"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_FILE="compose.yaml"
TARGET_VERSION=""
ASSUME_YES=0
DO_PULL=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --yes|-y)  ASSUME_YES=1; shift ;;
    --local|--no-pull) DO_PULL=0; shift ;;
    -h|--help) sed -n '2,24p' "$0"; exit 0 ;;
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
TARGET_REF="$(image_ref_for "${TARGET_VERSION}")"

# --- 1. a pinned version that is already deployed is a no-op ----------------
if [[ "${TARGET_VERSION}" == "${CURRENT_VERSION}" ]] && ! is_mutable_tag "${TARGET_VERSION}"; then
  log "already on the pinned version ${TARGET_VERSION}; nothing to do"
  exit 0
fi

log "current version: ${CURRENT_VERSION}  ->  target: ${TARGET_VERSION}"

# --- 2. fetch the target image ---------------------------------------------
IMAGE_ID_BEFORE="$(local_image_id "${TARGET_REF}")"

if [[ "${DO_PULL}" -eq 1 ]]; then
  log "pulling ${TARGET_REF}"
  if ! docker pull "${TARGET_REF}"; then
    if [[ -z "${IMAGE_ID_BEFORE}" ]]; then
      die "could not pull ${TARGET_REF} and it is not present locally; nothing was changed"
    fi
    warn "could not pull ${TARGET_REF}; falling back to the local copy"
  fi
fi

IMAGE_ID_AFTER="$(local_image_id "${TARGET_REF}")"
[[ -n "${IMAGE_ID_AFTER}" ]] || die "image ${TARGET_REF} is not available locally; nothing was changed"

# --- 3. mutable tag that did not move: leave the container alone ------------
if [[ "${TARGET_VERSION}" == "${CURRENT_VERSION}" ]]; then
  RUNNING_IMAGE_ID="$(container_image_id || true)"
  RUNNING_STATUS="$(container_health "$(container_id)" || echo unknown)"
  if [[ "${IMAGE_ID_AFTER}" == "${IMAGE_ID_BEFORE}" ]] \
     && [[ "${RUNNING_IMAGE_ID}" == "${IMAGE_ID_AFTER}" ]] \
     && [[ "${RUNNING_STATUS}" == "healthy" ]]; then
    log "already running the newest ${TARGET_VERSION} (${IMAGE_ID_AFTER:0:19}); nothing to do"
    exit 0
  fi
  if [[ "${IMAGE_ID_AFTER}" != "${IMAGE_ID_BEFORE}" ]]; then
    log "a newer ${TARGET_VERSION} image is available (${IMAGE_ID_BEFORE:0:19} -> ${IMAGE_ID_AFTER:0:19})"
  else
    log "image unchanged, but the running container needs to be (re)created"
  fi
fi

if [[ "${ASSUME_YES}" -ne 1 ]]; then
  printf 'Proceed? [y/N] '
  read -r answer
  [[ "${answer}" == "y" || "${answer}" == "Y" ]] || die "aborted by user"
fi

# --- 4. record the state we must be able to return to ----------------------
SCHEMA_BEFORE="$(schema_version 2>/dev/null || true)"
CATEGORIES_BEFORE="$(db_field categories 2>/dev/null || true)"
BOOKMARKS_BEFORE="$(db_field bookmarks 2>/dev/null || true)"
if [[ -n "${SCHEMA_BEFORE}" ]]; then
  log "database before update: schema=${SCHEMA_BEFORE} categories=${CATEGORIES_BEFORE:-?} bookmarks=${BOOKMARKS_BEFORE:-?}"
else
  warn "could not read the database before the update; rollback will restore the snapshot unconditionally"
fi

# --- 5. stop + snapshot -----------------------------------------------------
log "stopping the application for a consistent snapshot"
compose stop "${SERVICE_NAME}" || true

SNAPSHOT="$(snapshot_data_dir)"
log "snapshot written to ${SNAPSHOT}"

# --- 6. rollback ------------------------------------------------------------
rollback() {
  local reason="$1" schema_after status schema_now categories_now bookmarks_now
  warn "update failed: ${reason}"

  # Decide whether the previous image may be started against this database.
  # The check reads the database file, so it works without HTTP and in the NPM
  # mode; when it cannot be determined the snapshot is restored to be safe.
  schema_after="$(schema_version 2>/dev/null || true)"
  if [[ -z "${schema_after}" ]]; then
    warn "cannot determine the database schema version; restoring the pre-update snapshot before rolling back"
    restore_snapshot "${SNAPSHOT}"
  elif [[ -z "${SCHEMA_BEFORE}" ]]; then
    warn "the pre-update schema version is unknown; restoring the pre-update snapshot before rolling back"
    restore_snapshot "${SNAPSHOT}"
  elif [[ "${schema_after}" -gt "${SCHEMA_BEFORE}" ]]; then
    warn "the new build migrated the database (schema ${SCHEMA_BEFORE} -> ${schema_after});"
    warn "the previous image cannot read it, so the pre-update snapshot is restored first"
    restore_snapshot "${SNAPSHOT}"
  fi

  log "switching back to ${CURRENT_VERSION}"
  set_env APP_VERSION "${CURRENT_VERSION}"
  compose up -d || warn "compose up reported a failure during rollback"

  # Verify the rolled-back deployment: never trust the exit code alone.
  if ! status="$(wait_for_health 120)"; then
    warn "the rolled-back container is not healthy (status: ${status})"
    compose logs --tail=60 "${SERVICE_NAME}" || true
    warn "manual attention required. Pre-update snapshot: ${SNAPSHOT}"
    exit 1
  fi

  schema_now="$(schema_version 2>/dev/null || true)"
  categories_now="$(db_field categories 2>/dev/null || true)"
  bookmarks_now="$(db_field bookmarks 2>/dev/null || true)"

  if [[ -n "${SCHEMA_BEFORE}" && "${schema_now}" != "${SCHEMA_BEFORE}" ]]; then
    warn "rolled-back schema is ${schema_now}, expected ${SCHEMA_BEFORE}"
  fi
  if [[ -n "${BOOKMARKS_BEFORE}" && "${bookmarks_now}" != "${BOOKMARKS_BEFORE}" ]]; then
    warn "rolled-back bookmarks count is ${bookmarks_now}, expected ${BOOKMARKS_BEFORE}"
  fi

  log "rollback verified: version=${CURRENT_VERSION} status=${status} schema=${schema_now:-unknown} categories=${categories_now:-?} bookmarks=${bookmarks_now:-?}"
  warn "pre-update snapshot kept at: ${SNAPSHOT}"
  exit 1
}

# --- 7. switch version and start -------------------------------------------
log "switching APP_VERSION to ${TARGET_VERSION}"
set_env APP_VERSION "${TARGET_VERSION}"

log "starting the updated application"
if ! compose up -d; then
  rollback "container start failed"
fi

log "verifying health"
if ! status="$(wait_for_health 120)"; then
  compose logs --tail=80 "${SERVICE_NAME}" || true
  rollback "health check failed (status: ${status})"
fi

SCHEMA_AFTER="$(schema_version 2>/dev/null || true)"
if [[ -n "${SCHEMA_BEFORE}" && -n "${SCHEMA_AFTER}" ]]; then
  if [[ "${SCHEMA_AFTER}" -lt "${SCHEMA_BEFORE}" ]]; then
    rollback "schema version went backwards (${SCHEMA_BEFORE} -> ${SCHEMA_AFTER})"
  fi
  log "database schema version: ${SCHEMA_BEFORE} -> ${SCHEMA_AFTER}"
fi

CATEGORIES_AFTER="$(db_field categories 2>/dev/null || true)"
BOOKMARKS_AFTER="$(db_field bookmarks 2>/dev/null || true)"
if [[ -n "${BOOKMARKS_BEFORE}" && -n "${BOOKMARKS_AFTER}" && "${BOOKMARKS_AFTER}" != "${BOOKMARKS_BEFORE}" ]]; then
  warn "bookmark count changed during the update (${BOOKMARKS_BEFORE} -> ${BOOKMARKS_AFTER}); verify the result"
fi

log "update complete: ${CURRENT_VERSION} -> ${TARGET_VERSION} (status: ${status})"
log "data: schema=${SCHEMA_AFTER:-unknown} categories=${CATEGORIES_AFTER:-?} bookmarks=${BOOKMARKS_AFTER:-?}"
log "snapshot retained at: ${SNAPSHOT}"
log "to roll back manually: ./scripts/restore.sh ${SNAPSHOT} --version ${CURRENT_VERSION}"
