#!/usr/bin/env bash
#
# XiaoWork Nav — deploy / restart the application.
#
#   ./scripts/deploy.sh                 # pull the configured version and start
#   ./scripts/deploy.sh --build         # rebuild the image locally and start
#   ./scripts/deploy.sh --no-pull       # restart without pulling
#   ./scripts/deploy.sh --compose compose.npm.yaml
#   ./scripts/deploy.sh --logs          # follow the application logs
#
# Safe to re-run: it never removes volumes or data. For the first install use
# scripts/install.sh.

set -Eeuo pipefail

SCRIPT_NAME="deploy"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_FILE="compose.yaml"
BUILD_LOCAL=0
DO_PULL=1
SHOW_LOGS=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --build)   BUILD_LOCAL=1; DO_PULL=0; shift ;;
    --no-pull) DO_PULL=0; shift ;;
    --logs)    SHOW_LOGS=1; shift ;;
    -h|--help) sed -n '2,13p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

# shellcheck source=scripts/lib.sh
source "${SCRIPT_DIR}/lib.sh"

require_docker
require_env_file
validate_compose

# Make sure the container user can write to the bind-mounted data directory
# before touching anything else (see docs/installation.md §7).
ensure_data_dirs

if [[ "${SHOW_LOGS}" -eq 1 ]]; then
  compose logs -f "${SERVICE_NAME}"
  exit 0
fi

if [[ "${BUILD_LOCAL}" -eq 1 ]]; then
  log "building local image (xiaowork-nav:local)"
  docker build \
    --build-arg "VERSION=$(git -C "${ROOT_DIR}" describe --tags --always 2>/dev/null || echo dev)" \
    --build-arg "COMMIT=$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)" \
    --build-arg "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -t xiaowork-nav:local "${ROOT_DIR}"
  set_env XN_IMAGE xiaowork-nav
  set_env APP_VERSION local
elif [[ "${DO_PULL}" -eq 1 ]]; then
  log "pulling $(image_ref)"
  compose pull || warn "pull failed; using the locally available image if present"
fi

log "starting services"
compose up -d

log "waiting for health"
if ! status="$(wait_for_health 120)"; then
  compose logs --tail=60 "${SERVICE_NAME}" || true
  die "health check failed (status: ${status})"
fi

log "deployed $(image_ref) (status: ${status})"
compose ps
