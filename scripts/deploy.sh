#!/usr/bin/env bash
#
# XiaoWork Nav — deploy / (re)start the application.
#
# Safe to re-run: it never removes volumes or data.
#
# Usage:
#   ./scripts/deploy.sh [--compose compose.npm.yaml] [--build] [--no-pull]
#
# Modes:
#   default   pull the image referenced by XN_IMAGE:APP_VERSION and start
#   --build   build the image locally (tag: xiaowork-nav:local) and start

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"

COMPOSE_FILE="compose.yaml"
BUILD_LOCAL=0
DO_PULL=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose) COMPOSE_FILE="${2:?missing value for --compose}"; shift 2 ;;
    --build)   BUILD_LOCAL=1; shift ;;
    --no-pull) DO_PULL=0; shift ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

log()  { printf '\033[1;34m[deploy]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }

command -v docker >/dev/null 2>&1 || die "docker is required but was not found"
docker compose version >/dev/null 2>&1 || die "the Docker Compose plugin (docker compose) is required"

ENV_FILE="${DEPLOY_DIR}/.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f "${DEPLOY_DIR}/.env.example" ]]; then
    cp "${DEPLOY_DIR}/.env.example" "${ENV_FILE}"
    die "created ${ENV_FILE} from .env.example — edit it, then re-run this script"
  fi
  die "missing ${ENV_FILE}"
fi

COMPOSE_PATH="${DEPLOY_DIR}/${COMPOSE_FILE}"
[[ -f "${COMPOSE_PATH}" ]] || die "compose file not found: ${COMPOSE_PATH}"

compose() {
  docker compose -f "${COMPOSE_PATH}" --env-file "${ENV_FILE}" "$@"
}

log "validating compose configuration (${COMPOSE_FILE})"
compose config >/dev/null || die "compose configuration is invalid"

if [[ "${BUILD_LOCAL}" -eq 1 ]]; then
  log "building local image (xiaowork-nav:local)"
  docker build -t xiaowork-nav:local \
    --build-arg "VERSION=local" \
    --build-arg "COMMIT=$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)" \
    --build-arg "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    "${ROOT_DIR}"
  if grep -qE '^XN_IMAGE=' "${ENV_FILE}"; then
    sed -i.bak 's|^XN_IMAGE=.*|XN_IMAGE=xiaowork-nav|' "${ENV_FILE}" && rm -f "${ENV_FILE}.bak"
  else
    printf 'XN_IMAGE=xiaowork-nav\n' >> "${ENV_FILE}"
  fi
  if grep -qE '^APP_VERSION=' "${ENV_FILE}"; then
    sed -i.bak 's|^APP_VERSION=.*|APP_VERSION=local|' "${ENV_FILE}" && rm -f "${ENV_FILE}.bak"
  else
    printf 'APP_VERSION=local\n' >> "${ENV_FILE}"
  fi
elif [[ "${DO_PULL}" -eq 1 ]]; then
  log "pulling image"
  compose pull || warn "pull failed; continuing with the local image if present"
fi

log "starting services"
compose up -d

log "waiting for the app to become healthy"
cid="$(compose ps -q app)"
[[ -n "${cid}" ]] || die "app container was not created"

deadline=$((SECONDS + 90))
status="unknown"
while [[ ${SECONDS} -lt ${deadline} ]]; do
  status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${cid}" 2>/dev/null || echo unknown)"
  [[ "${status}" == "healthy" || "${status}" == "running" ]] && break
  [[ "${status}" == "unhealthy" ]] && break
  sleep 3
done

if [[ "${status}" == "unhealthy" ]]; then
  warn "container reported unhealthy; recent logs:"
  compose logs --tail=40 app || true
  die "deployment health check failed"
fi

log "deployment complete (status: ${status})"
log "local health check:"
if command -v curl >/dev/null 2>&1; then
  # Best-effort check against the published port if configured.
  port="$(grep -E '^XN_PUBLISH_PORT=' "${ENV_FILE}" | tail -n1 | cut -d= -f2 || true)"
  if [[ -n "${port}" ]]; then
    curl -fsS "http://127.0.0.1:${port}/healthz" && echo || warn "could not reach http://127.0.0.1:${port}/healthz (may be NPM mode)"
  fi
fi
compose ps
