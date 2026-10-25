#!/usr/bin/env bash
#
# Shared helpers for the XiaoWork Nav deployment scripts.
#
# Callers must set these before sourcing:
#   ROOT_DIR    repository root
#   DEPLOY_DIR  directory holding compose files and .env
#
# Optional:
#   ENV_FILE      defaults to ${DEPLOY_DIR}/.env
#   COMPOSE_FILE  defaults to compose.yaml
#   SERVICE_NAME  defaults to app
#
# This file is sourced, never executed.

set -Eeuo pipefail

: "${ROOT_DIR:?ROOT_DIR must be set before sourcing lib.sh}"
: "${DEPLOY_DIR:?DEPLOY_DIR must be set before sourcing lib.sh}"
ENV_FILE="${ENV_FILE:-${DEPLOY_DIR}/.env}"
COMPOSE_FILE="${COMPOSE_FILE:-compose.yaml}"
SERVICE_NAME="${SERVICE_NAME:-app}"
DEFAULT_IMAGE="ghcr.io/xiaowork-dev/xiaowork-nav"

# ---------- logging ----------

log()  { printf '\033[1;34m[%s]\033[0m %s\n' "${SCRIPT_NAME:-xiaowork-nav}" "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

# ---------- preconditions ----------

require_docker() {
  command -v docker >/dev/null 2>&1 || die "docker is required but was not found"
  docker compose version >/dev/null 2>&1 || die "the Docker Compose plugin (docker compose) is required"
  docker info >/dev/null 2>&1 || die "cannot talk to the Docker daemon; is it running?"
}

require_env_file() {
  [[ -f "${ENV_FILE}" ]] || die "missing ${ENV_FILE}; run scripts/install.sh first"
}

ensure_env_file() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    [[ -f "${DEPLOY_DIR}/.env.example" ]] || die "missing ${ENV_FILE} and ${DEPLOY_DIR}/.env.example"
    cp "${DEPLOY_DIR}/.env.example" "${ENV_FILE}"
    log "created ${ENV_FILE} from .env.example"
  fi
}

# ---------- compose ----------

compose_file_path() { printf '%s/%s\n' "${DEPLOY_DIR}" "${COMPOSE_FILE}"; }

compose() {
  docker compose -f "$(compose_file_path)" --env-file "${ENV_FILE}" "$@"
}

validate_compose() {
  [[ -f "$(compose_file_path)" ]] || die "compose file not found: $(compose_file_path)"
  compose config >/dev/null || die "compose configuration is invalid ($(compose_file_path))"
}

# ---------- .env access ----------

get_env() {
  local key="$1"
  grep -E "^${key}=" "${ENV_FILE}" 2>/dev/null | tail -n1 | cut -d= -f2- || true
}

# set_env writes KEY=VALUE atomically, preserving the rest of the file.
set_env() {
  local key="$1" value="$2" tmp
  tmp="$(mktemp)"
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    awk -v k="${key}" -v v="${value}" 'BEGIN{FS=OFS="="} $1==k{ $0=k"="v } {print}' "${ENV_FILE}" > "${tmp}"
  else
    cat "${ENV_FILE}" > "${tmp}"
    printf '%s=%s\n' "${key}" "${value}" >> "${tmp}"
  fi
  mv "${tmp}" "${ENV_FILE}"
}

# data_host_dir resolves XN_DATA_HOST_DIR to an absolute path.
data_host_dir() {
  local dir
  dir="$(get_env XN_DATA_HOST_DIR)"
  dir="${dir:-./data}"
  case "${dir}" in
    /*)                       printf '%s\n' "${dir}" ;;           # POSIX absolute
    [A-Za-z]:/*|[A-Za-z]:\\*) printf '%s\n' "${dir}" ;;           # Windows absolute
    *)                        printf '%s\n' "${DEPLOY_DIR}/${dir#./}" ;;
  esac
}

# dir_owner prints the numeric uid that owns a path (empty when unknown).
dir_owner() {
  stat -c '%u' "$1" 2>/dev/null || stat -f '%u' "$1" 2>/dev/null || printf ''
}

# ensure_data_dirs creates the persistent directories and makes sure the
# container user can write to them.
#
# The container never runs as root, so the fix is never "chmod 777": either the
# directory is given to the container user, or the container is told to run as
# the directory's owner through XN_UID/XN_GID.
ensure_data_dirs() {
  local data_dir uploads backups uid gid had_data=0
  data_dir="$(data_host_dir)"
  uploads="${data_dir}/uploads"
  backups="${data_dir}/backups"

  # Decide whether this is a fresh directory *before* creating subdirectories,
  # because creating them would make an empty directory look non-empty.
  if [[ -d "${data_dir}" && -n "$(ls -A "${data_dir}" 2>/dev/null)" ]]; then
    had_data=1
  fi

  mkdir -p "${data_dir}" "${uploads}" "${backups}" 2>/dev/null \
    || die "cannot create the data directory: ${data_dir}"

  uid="$(get_env XN_UID)"; uid="${uid:-100}"
  gid="$(get_env XN_GID)"; gid="${gid:-101}"

  # Already owned by the container user: nothing to do.
  if [[ "$(dir_owner "${data_dir}")" == "${uid}" ]]; then
    return 0
  fi

  if [[ "$(id -u)" -eq 0 ]]; then
    log "giving ${data_dir} to the container user (uid ${uid}:${gid})"
    chown -R "${uid}:${gid}" "${data_dir}" || warn "chown failed; the container may not be able to write"
    return 0
  fi

  if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
    log "giving ${data_dir} to the container user (uid ${uid}:${gid}) via sudo"
    sudo chown -R "${uid}:${gid}" "${data_dir}" || warn "chown failed; the container may not be able to write"
    return 0
  fi

  # Ownership cannot be changed here.
  #
  # Only a *fresh* install may re-point the container at this host account:
  # switching on an existing deployment would make the container unable to
  # write the files it already created as the default user. Existing data is
  # therefore left exactly as it is, with an actionable warning.
  if [[ "${had_data}" -eq 0 ]]; then
    local host_uid host_gid
    host_uid="$(id -u)"
    host_gid="$(id -g)"
    warn "cannot change the owner of ${data_dir}; running the container as uid ${host_uid}:${host_gid} instead"
    set_env XN_UID "${host_uid}"
    set_env XN_GID "${host_gid}"
    return 0
  fi

  warn "${data_dir} is not owned by the container user (uid ${uid}:${gid})."
  warn "if the container reports a permission error, fix it with one of:"
  warn "  sudo chown -R ${uid}:${gid} ${data_dir}"
  warn "  set XN_UID/XN_GID in deploy/.env to the directory owner (id -u / id -g)"
}

# ---------- image references ----------

image_repo() {
  local repo
  repo="$(get_env XN_IMAGE)"
  printf '%s\n' "${repo:-${DEFAULT_IMAGE}}"
}

# image_ref_for <tag> builds "<repo>:<tag>"; without an argument the configured
# APP_VERSION is used.
image_ref_for() {
  local tag="${1:-$(get_env APP_VERSION)}"
  tag="${tag:-latest}"
  printf '%s:%s\n' "$(image_repo)" "${tag}"
}

image_ref() { image_ref_for "$(get_env APP_VERSION)"; }

local_image_id() {
  docker image inspect --format '{{.Id}}' "$1" 2>/dev/null || true
}

container_image_id() {
  local cid
  cid="$(container_id)"
  [[ -n "${cid}" ]] || return 1
  docker inspect --format '{{.Image}}' "${cid}" 2>/dev/null
}

# A tag is mutable when it can move to a new digest: "latest" or a partial
# version such as "v2.0". A full "vMAJOR.MINOR.PATCH" is treated as immutable.
is_mutable_tag() {
  local tag="$1"
  if [[ -z "${tag}" || "${tag}" == "latest" ]]; then
    return 0
  fi
  if [[ "${tag}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    return 1
  fi
  return 0
}

# ---------- tar ----------

# GNU tar interprets "host:path" in -f/-C arguments (which breaks Windows-style
# paths such as C:/data). --force-local disables that; other tar implementations
# do not need it and do not understand it.
TAR_EXTRA=""
if tar --version 2>/dev/null | grep -qi 'GNU tar'; then
  TAR_EXTRA="--force-local"
fi

# ---------- runtime helpers ----------

container_id() {
  compose ps -q "${SERVICE_NAME}" 2>/dev/null || true
}

container_health() {
  local cid="$1"
  docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${cid}" 2>/dev/null || echo unknown
}

# wait_for_health waits until the service is healthy/running.
# Prints the final status; returns 0 on healthy/running, 1 otherwise.
wait_for_health() {
  local timeout="${1:-120}" cid status deadline
  cid="$(container_id)"
  [[ -n "${cid}" ]] || { echo "no-container"; return 1; }

  deadline=$((SECONDS + timeout))
  while [[ ${SECONDS} -lt ${deadline} ]]; do
    status="$(container_health "${cid}")"
    case "${status}" in
      healthy|running) echo "${status}"; return 0 ;;
      unhealthy|exited|dead) echo "${status}"; return 1 ;;
    esac
    sleep 3
  done
  echo "timeout"
  return 1
}

# ---------- database inspection ----------
#
# The scripts must be able to read the database state in every Compose mode and
# even when the application cannot start. Asking the running container over HTTP
# fails in the NPM mode (no published host port) and when the container is down,
# so the application image is run as a one-shot helper against the data
# directory instead.

# db_info prints key=value lines describing the database, or returns 1 when the
# image is unavailable or the database cannot be read.
db_info() {
  local data_dir image out
  data_dir="$(data_host_dir)"
  image="$(image_ref)"
  [[ -d "${data_dir}" ]] || return 1
  docker image inspect "${image}" >/dev/null 2>&1 || return 1

  # MSYS_NO_PATHCONV/MSYS2_ARG_CONV_EXCL stop Git Bash on Windows from rewriting
  # the in-container "/data/app.db" argument into a Windows path. They are inert
  # on Linux and macOS.
  out="$(MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*' docker run --rm \
    -v "${data_dir}:/data" \
    "${image}" --db-info --db /data/app.db 2>/dev/null || true)"
  [[ -n "${out}" ]] || return 1
  printf '%s\n' "${out}"
}

# db_field prints a single field from db_info, or returns 1 when unavailable.
db_field() {
  local key="$1" info value
  info="$(db_info 2>/dev/null || true)"
  [[ -n "${info}" ]] || return 1
  value="$(printf '%s\n' "${info}" | sed -n "s/^${key}=//p" | tail -n1)"
  [[ -n "${value}" ]] || return 1
  printf '%s\n' "${value}"
}

schema_version() { db_field schema_version; }

# ---------- snapshots ----------

snapshot_dir() {
  printf '%s\n' "$(dirname "$(data_host_dir)")/xiaowork-nav-backups"
}

# snapshot_data_dir takes a consistent snapshot of the data directory.
# The caller is responsible for stopping the service first.
snapshot_data_dir() {
  local data_dir backup_dir stamp target
  data_dir="$(data_host_dir)"
  [[ -d "${data_dir}" ]] || die "data directory not found: ${data_dir}"
  backup_dir="$(snapshot_dir)"
  mkdir -p "${backup_dir}"
  stamp="$(date -u +%Y%m%d-%H%M%S)"
  target="${backup_dir}/pre-update-$(get_env APP_VERSION)-${stamp}.tar.gz"
  # shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
  tar ${TAR_EXTRA} -czf "${target}" -C "${data_dir}" . || die "snapshot failed"
  printf '%s\n' "${target}"
}

# restore_snapshot replaces the data directory with the snapshot contents.
# The caller must stop the service first.
restore_snapshot() {
  local snapshot="$1" data_dir
  data_dir="$(data_host_dir)"
  log "restoring data from ${snapshot}"
  rm -rf "${data_dir:?}"/* 2>/dev/null || true
  # shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
  tar ${TAR_EXTRA} -xzf "${snapshot}" -C "${data_dir}" || die "failed to restore snapshot ${snapshot}"
}

# version_from_snapshot infers the application version from a snapshot filename
# such as backup-v2.0.0-20261025-101500.tar.gz. It prints nothing when the name
# does not follow the "<kind>-<version>-<timestamp>.tar.gz" convention, so the
# caller can refuse to start instead of guessing.
version_from_snapshot() {
  local base
  base="$(basename "$1")"
  if [[ "${base}" =~ ^(backup|pre-update|restore)-(.+)-[0-9]{8}-[0-9]{6}(-[0-9]+)?\.tar\.gz$ ]]; then
    printf '%s\n' "${BASH_REMATCH[2]}"
  fi
}
