#!/usr/bin/env bash
#
# XiaoWork Nav — one-command install.
#
#   ./scripts/install.sh                    # pull the published image and start
#   ./scripts/install.sh --build            # build the image locally and start
#   ./scripts/install.sh --compose compose.npm.yaml
#   ./scripts/install.sh --port 8787 --bind 127.0.0.1
#
# Safe to re-run: it never deletes data or volumes.

set -Eeuo pipefail

SCRIPT_NAME="install"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
COMPOSE_FILE="compose.yaml"
BUILD_LOCAL=0
DO_PULL=1
BIND_ADDR=""
PUBLISH_PORT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --build)   BUILD_LOCAL=1; DO_PULL=0; shift ;;
    --no-pull) DO_PULL=0; shift ;;
    --bind)    BIND_ADDR="${2:?missing value for --bind}"; shift 2 ;;
    --port)    PUBLISH_PORT="${2:?missing value for --port}"; shift 2 ;;
    -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

# shellcheck source=scripts/lib.sh
source "${SCRIPT_DIR}/lib.sh"

log "checking prerequisites"
require_docker
ensure_env_file
if [[ -n "${BIND_ADDR}" ]]; then set_env XN_BIND_ADDR "${BIND_ADDR}"; fi
if [[ -n "${PUBLISH_PORT}" ]]; then set_env XN_PUBLISH_PORT "${PUBLISH_PORT}"; fi

log "validating compose configuration (${COMPOSE_FILE})"
validate_compose

if [[ "${BUILD_LOCAL}" -eq 1 ]]; then
  log "building image locally (xiaowork-nav:local)"
  docker build \
    --build-arg "VERSION=$(git -C "${ROOT_DIR}" describe --tags --always 2>/dev/null || echo dev)" \
    --build-arg "COMMIT=$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)" \
    --build-arg "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -t xiaowork-nav:local "${ROOT_DIR}"
  set_env XN_IMAGE xiaowork-nav
  set_env APP_VERSION local
elif [[ "${DO_PULL}" -eq 1 ]]; then
  log "pulling $(image_ref)"
  compose pull || warn "pull failed; continuing with a locally available image if present"
fi

log "starting services"
compose up -d

log "waiting for the application to become healthy"
if ! status="$(wait_for_health 120)"; then
  warn "recent container logs:"
  compose logs --tail=60 "${SERVICE_NAME}" || true
  die "health check failed (status: ${status})"
fi
log "container status: ${status}"

data_dir="$(data_host_dir)"
cat <<EOF

XiaoWork Nav is running.

  image        $(image_ref)
  data dir     ${data_dir}
  bind         $(get_env XN_BIND_ADDR):$(get_env XN_PUBLISH_PORT)  (compose file: ${COMPOSE_FILE})

Next steps
  1. Open the site. If XN_BIND_ADDR is 127.0.0.1, use an SSH tunnel or your
     reverse proxy; the app is intentionally not exposed publicly.
  2. Complete the first-run wizard to create the administrator account.
  3. Import existing data from Admin -> Data management.

Useful commands
  logs      ./scripts/deploy.sh --logs      (or: docker compose -f deploy/${COMPOSE_FILE} logs -f app)
  update    ./scripts/update.sh vX.Y.Z
  backup    ./scripts/backup.sh
  restore   ./scripts/restore.sh <snapshot.tar.gz>

EOF
