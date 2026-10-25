#!/usr/bin/env bash
#
# Shared helpers for the XiaoWork Nav deployment scripts.
#
# Callers must set these before sourcing:
#   ROOT_DIR    repository root
#   DEPLOY_DIR  directory holding compose files and .env
#   ENV_FILE    path to the .env file
#   COMPOSE_FILE  compose file name (default compose.yaml)
#
# This file is sourced, never executed.

set -Eeuo pipefail

: "${ROOT_DIR:?ROOT_DIR must be set before sourcing lib.sh}"
: "${DEPLOY_DIR:?DEPLOY_DIR must be set before sourcing lib.sh}"
ENV_FILE="${ENV_FILE:-${DEPLOY_DIR}/.env}"
COMPOSE_FILE="${COMPOSE_FILE:-compose.yaml}"
SERVICE_NAME="${SERVICE_NAME:-app}"

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

image_ref() {
  local image version
  image="$(get_env XN_IMAGE)"
  version="$(get_env APP_VERSION)"
  image="${image:-ghcr.io/xiaowork-dev/xiaowork-nav}"
  version="${version:-latest}"
  printf '%s:%s\n' "${image}" "${version}"
}

# ---------- tar helpers ----------

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

# app_schema_version reads the database schema version from the running app.
app_schema_version() {
  local port version_json
  port="$(get_env XN_PUBLISH_PORT)"
  port="${port:-8787}"
  version_json="$(curl -fsS --max-time 5 "http://127.0.0.1:${port}/api/v1/version" 2>/dev/null || true)"
  [[ -n "${version_json}" ]] || return 1
  printf '%s\n' "${version_json}" | sed -n 's/.*"schema_version":\([0-9]*\).*/\1/p'
}

# snapshot_data_dir takes a consistent snapshot of the data directory.
# The caller is responsible for stopping the service first.
snapshot_data_dir() {
  local data_dir backup_dir stamp target
  data_dir="$(data_host_dir)"
  [[ -d "${data_dir}" ]] || die "data directory not found: ${data_dir}"
  backup_dir="$(dirname "${data_dir}")/xiaowork-nav-backups"
  mkdir -p "${backup_dir}"
  stamp="$(date -u +%Y%m%d-%H%M%S)"
  target="${backup_dir}/pre-update-$(get_env APP_VERSION)-${stamp}.tar.gz"
  # shellcheck disable=SC2086  # TAR_EXTRA is intentionally word-split (may be empty)
  tar ${TAR_EXTRA} -czf "${target}" -C "${data_dir}" . || die "snapshot failed"
  printf '%s\n' "${target}"
}
