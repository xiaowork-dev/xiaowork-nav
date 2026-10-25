#!/usr/bin/env bash
#
# Fault-injection tests for the deployment scripts.
#
# Requires Docker. Creates an isolated deployment under a temporary directory,
# builds throwaway images, and drives scripts/update.sh and scripts/restore.sh
# through the happy path and through the failure paths that matter:
#
#   1. database inspection reads the database file (any Compose mode)
#   2. a pinned version that is already deployed is a no-op
#   3. "latest" with an unchanged image does not recreate the container
#   4. an unavailable target image changes nothing
#   5. a failed image pull with no local copy changes nothing
#   6. a container that fails to start rolls back (schema unchanged)
#   7. a health failure *after* a migration restores the snapshot first
#   8. a new image that cannot report the database version restores the
#      snapshot (fail-safe: never start an old image against a new schema)
#   9. the schema version is readable while the application is stopped
#      (the NPM mode publishes no host port, so HTTP cannot be used)
#  10. restore.sh switches to the snapshot's version before starting
#  11. restore.sh refuses to start when the snapshot version is unknown
#  12. an NPM-style deployment (external network, no host port) rolls back and
#      is verified after a failed upgrade
#  13. upgrading from the released v2.0.1 advances the schema while keeping the
#      data, the manual SVG icon and the upload metadata
#  14. an unwritable data directory fails fast with actionable guidance
#
# Usage: ./scripts/tests/deploy-faults.test.sh
#   XN_TEST_GOPROXY  Go module proxy for the image builds (default: upstream)
#   XN_TEST_PORT     host port used by the baseline deployment (default: 18999)
#
# The tests never touch a real deployment: everything happens under a temporary
# data directory and a temporary .env, both removed on exit.

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
WORK_ROOT="${TMPDIR:-/tmp}/xiaowork-nav-deploy-tests"

PORT="${XN_TEST_PORT:-18999}"
DATA_DIR="${WORK_ROOT}/data"
# Keep in sync with backend/internal/migrations/sql (highest NNNN_*.up.sql).
EXPECTED_SCHEMA=3
# Docker needs a host path it understands; on Windows (Git Bash) that means a
# C:/... style path rather than the MSYS /tmp/... view.
if command -v cygpath >/dev/null 2>&1; then
  DATA_DIR_HOST="$(cygpath -m "${DATA_DIR}")"
  WORK_ROOT_HOST="$(cygpath -m "${WORK_ROOT}")"
else
  DATA_DIR_HOST="${DATA_DIR}"
  WORK_ROOT_HOST="${WORK_ROOT}"
fi
BACKUP_DIR="$(dirname "${DATA_DIR_HOST}")/xiaowork-nav-backups"

TESTFAIL_COMPOSE="compose.testfail.local.yaml"
NPMTEST_COMPOSE="compose.npmtest.local.yaml"
NPM_TEST_NETWORK="xn-deploy-test-net"

ENV_BACKUP=""
LATEST_BACKUP_ID=""
FAILED=0
SCENARIO_FAILED=0
CURRENT_SCENARIO=""

# ---------- reporting ----------

c_reset=$'\033[0m'; c_green=$'\033[1;32m'; c_red=$'\033[1;31m'; c_blue=$'\033[1;34m'

scenario() { CURRENT_SCENARIO="$1"; SCENARIO_FAILED=0; printf '\n%s== %s%s\n' "${c_blue}" "$1" "${c_reset}"; }
ok()       { printf '  %sPASS%s %s\n' "${c_green}" "${c_reset}" "$1"; }
bad()      { printf '  %sFAIL%s %s\n' "${c_red}" "${c_reset}" "$1"; FAILED=1; SCENARIO_FAILED=1; }

# dump prints captured command output when the current scenario failed.
dump() {
  if [[ "${SCENARIO_FAILED}" -eq 1 ]]; then
    printf '  --- captured output ---\n%s\n  -----------------------\n' "$1"
  fi
}

assert_contains() { # <haystack> <needle> <description>
  if printf '%s' "$1" | grep -qF -- "$2"; then ok "$3"; else bad "$3 (missing: $2)"; fi
}
assert_eq() { # <actual> <expected> <description>
  if [[ "$1" == "$2" ]]; then ok "$3"; else bad "$3 (got '$1', want '$2')"; fi
}
assert_not_eq() {
  if [[ "$1" != "$2" ]]; then ok "$3"; else bad "$3 (both '$1')"; fi
}

# ---------- environment ----------

cleanup() {
  compose down -v >/dev/null 2>&1 || true
  docker network rm "${NPM_TEST_NETWORK}" >/dev/null 2>&1 || true
  for img in xiaowork-nav:v2.0.0 xiaowork-nav:v2.0.1 xiaowork-nav:v2.0.1-old xiaowork-nav:v2.0.2 \
             xiaowork-nav:test-broken xiaowork-nav:latest; do
    docker rmi "$img" >/dev/null 2>&1 || true
  done
  # Put back a "latest" tag that existed before the tests ran.
  if [[ -n "${LATEST_BACKUP_ID}" ]]; then
    docker tag "${LATEST_BACKUP_ID}" xiaowork-nav:latest >/dev/null 2>&1 || true
  fi
  rm -f "${DEPLOY_DIR}/${TESTFAIL_COMPOSE}" "${DEPLOY_DIR}/${NPMTEST_COMPOSE}"
  if [[ -n "${ENV_BACKUP}" && -f "${ENV_BACKUP}" ]]; then
    mv "${ENV_BACKUP}" "${DEPLOY_DIR}/.env"
  else
    rm -f "${DEPLOY_DIR}/.env"
  fi
  rm -rf "${WORK_ROOT}"
}
trap cleanup EXIT

# write_env <current_version> [bad_start_version] [bad_health_version] [proxy_network]
#
# <current_version> is what APP_VERSION is set to *before* the update runs; the
# update scripts then switch it to the target themselves.
write_env() {
  cat > "${DEPLOY_DIR}/.env" <<EOF
XN_IMAGE=xiaowork-nav
APP_VERSION=$1
XN_CONTAINER_NAME=xiaowork-nav-deploytest
XN_DATA_HOST_DIR=${DATA_DIR_HOST}
XN_BIND_ADDR=127.0.0.1
XN_PUBLISH_PORT=${PORT}
XN_PROXY_NETWORK=${4:-}
XN_LOG_LEVEL=warn
XN_ALLOW_GUEST_ACCESS=true
XN_SECURE_COOKIE=false
XN_BACKUP_KEEP=5
XN_UPDATE_REPO=
XN_TRUSTED_PROXIES=
XN_ALLOWED_ORIGINS=
XN_TEST_BAD_START=${2:-}
XN_TEST_BAD_HEALTH=${3:-}
TZ=UTC
EOF
}

# write_fail_compose <path> [proxy_network]
#
# A Compose file that publishes no host port (like the NPM variant) and can
# inject a failure that depends on which image is being started, so the same
# file can be used for the failing upgrade *and* for the rollback:
#   * XN_TEST_BAD_START  — that version's entrypoint exits immediately
#   * XN_TEST_BAD_HEALTH — that version never passes its health check
write_fail_compose() {
  local path="$1" net="${2:-}"
  cat > "${path}" <<'YAML'
name: xiaowork-nav

services:
  app:
    image: ${XN_IMAGE:-xiaowork-nav}:${APP_VERSION:-latest}
    container_name: ${XN_CONTAINER_NAME:-xiaowork-nav}
    restart: "no"
    env_file:
      - .env
    environment:
      XN_DATA_DIR: /data
      XN_HOST: 0.0.0.0
      XN_PORT: "8080"
      XN_LOG_LEVEL: warn
      XN_SECURE_COOKIE: "false"
      XN_TEST_APP_VERSION: ${APP_VERSION:-unknown}
    entrypoint:
      - /bin/sh
      - -c
      - 'if [ -n "$$XN_TEST_BAD_START" ] && [ "$$XN_TEST_APP_VERSION" = "$$XN_TEST_BAD_START" ]; then echo "injected start failure" >&2; exit 1; fi; exec /usr/local/bin/xiaowork-nav'
    volumes:
      - ${XN_DATA_HOST_DIR:-./data}:/data
    healthcheck:
      test:
        - CMD
        - /bin/sh
        - -c
        - '[ -z "$$XN_TEST_BAD_HEALTH" ] || [ "$$XN_TEST_APP_VERSION" != "$$XN_TEST_BAD_HEALTH" ]'
      interval: 2s
      timeout: 2s
      retries: 2
      start_period: 1s
YAML
  if [[ -n "${net}" ]]; then
    cat >> "${path}" <<YAML
    networks:
      - proxy

networks:
  proxy:
    external: true
    name: ${net}
YAML
  fi
}

# shellcheck source=scripts/lib.sh
source "${ROOT_DIR}/scripts/lib.sh"

base_url="http://127.0.0.1:${PORT}"

# login_cookie logs in as the seeded administrator and prints the session cookie.
login_cookie() {
  curl -fsS -i -X POST "${base_url}/api/v1/auth/login" -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"deploy-test-pass"}' \
    | tr -d '\r' | sed -n 's/^[Ss]et-[Cc]ookie: \([^;]*\).*/\1/p' | head -1
}

wait_http() { # <timeout seconds>
  local deadline=$((SECONDS + $1))
  while [[ ${SECONDS} -lt ${deadline} ]]; do
    if curl -fsS --max-time 3 "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then return 0; fi
    sleep 2
  done
  return 1
}

# ensure_seeded creates one category and one bookmark, once per data directory.
ensure_seeded() {
  local cats
  cats="$(db_field categories 2>/dev/null || echo 0)"
  if [[ "${cats}" =~ ^[0-9]+$ ]] && [[ "${cats}" -ge 1 ]]; then return 0; fi

  local cookie
  curl -fsS -X POST "${base_url}/api/v1/setup" -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"deploy-test-pass"}' >/dev/null 2>&1 || true
  cookie="$(login_cookie)"
  [[ -n "${cookie}" ]] || { bad "could not log in to seed data"; return 1; }
  curl -fsS -X POST "${base_url}/api/v1/admin/categories" -H 'Content-Type: application/json' \
    -H "Cookie: ${cookie}" -d '{"name":"Seed"}' >/dev/null
  curl -fsS -X POST "${base_url}/api/v1/admin/bookmarks" -H 'Content-Type: application/json' \
    -H "Cookie: ${cookie}" \
    -d '{"category_id":1,"title":"Seed Site","url":"https://example.com/"}' >/dev/null
  ok "seeded one category and one bookmark"
}

build_images() {
  log "building test images (this takes a few minutes)"
  local goproxy="${XN_TEST_GOPROXY:-https://proxy.golang.org,direct}"

  docker build --build-arg "GOPROXY=${goproxy}" \
    --build-arg VERSION=v2.0.0 --build-arg COMMIT=testbase --build-arg BUILD_TIME=test \
    -t xiaowork-nav:v2.0.0 "${ROOT_DIR}" >/dev/null
  docker tag xiaowork-nav:v2.0.0 xiaowork-nav:latest

  docker build --build-arg "GOPROXY=${goproxy}" \
    --build-arg VERSION=v2.0.1 --build-arg COMMIT=testnext --build-arg BUILD_TIME=test \
    -t xiaowork-nav:v2.0.1 "${ROOT_DIR}" >/dev/null

  # A second image that carries one extra migration, so an update can advance
  # the schema and the "migrated then failed" rollback path can be tested.
  local copy="${WORK_ROOT}/repo-mig"
  mkdir -p "${copy}"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --exclude .git --exclude node_modules --exclude dist "${ROOT_DIR}/" "${copy}/"
  else
    tar -C "${ROOT_DIR}" --exclude=./.git --exclude=./node_modules --exclude=./frontend/dist \
      -cf - . | tar -C "${copy}" -xf -
  fi
  cat > "${copy}/backend/internal/migrations/sql/9001_deploy_test.up.sql" <<'SQL'
ALTER TABLE categories ADD COLUMN deploy_test_marker TEXT NOT NULL DEFAULT '';
SQL
  printf 'ALTER TABLE categories DROP COLUMN deploy_test_marker;\n' \
    > "${copy}/backend/internal/migrations/sql/9001_deploy_test.down.sql"
  docker build --build-arg "GOPROXY=${goproxy}" \
    --build-arg VERSION=v2.0.2 --build-arg COMMIT=testmig --build-arg BUILD_TIME=test \
    -t xiaowork-nav:v2.0.2 "${copy}" >/dev/null

  # The previously released version, built from its own git tag, so the upgrade
  # scenario really starts on the old schema instead of a relabelled new build.
  local old="${WORK_ROOT}/repo-v2.0.1"
  mkdir -p "${old}"
  if git -C "${ROOT_DIR}" rev-parse -q --verify "refs/tags/v2.0.1" >/dev/null 2>&1; then
    git -C "${ROOT_DIR}" archive v2.0.1 | tar -C "${old}" -xf -
    docker build --build-arg "GOPROXY=${goproxy}" \
      --build-arg VERSION=v2.0.1 --build-arg COMMIT=release201 --build-arg BUILD_TIME=test \
      -t xiaowork-nav:v2.0.1-old "${old}" >/dev/null
  else
    warn "git tag v2.0.1 not found; skipping the released-version image"
  fi

  # An image that cannot start at all (and cannot report the database version).
  docker build -t xiaowork-nav:test-broken - <<'DOCKER' >/dev/null
FROM alpine:3.20
ENTRYPOINT ["/bin/false"]
DOCKER
  log "images built"
}

baseline_up() { # <app_version>
  write_env "$1"
  compose up -d >/dev/null
  if ! wait_for_health 120 >/dev/null; then
    bad "baseline ${1} did not become healthy"
    compose logs --tail=40 app || true
    return 1
  fi
  return 0
}

# ---------- scenarios ----------

scenario_db_inspection() {
  scenario "1. database inspection reads the file with the app running"
  baseline_up v2.0.0 || return

  local raw
  raw="$(db_info 2>&1 || true)"
  printf '  data_dir: %s\n' "$(data_host_dir)"
  printf '  image:    %s\n' "$(image_ref)"
  printf '  db_info:  %s\n' "$(printf '%s' "${raw}" | tr '\n' ' ')"
  assert_contains "${raw}" "exists=true" "db_info reports an existing database"
  assert_eq "$(db_field schema_version 2>/dev/null || true)" "${EXPECTED_SCHEMA}" "schema version read from the file"
}

scenario_pinned_noop() {
  scenario "2. pinned version already deployed is a no-op"
  baseline_up v2.0.0 || return

  local out
  out="$("${ROOT_DIR}/scripts/update.sh" v2.0.0 --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "already on the pinned version" "reports a no-op"
}

scenario_latest_unchanged() {
  scenario "3. latest with an unchanged image does not recreate the container"
  baseline_up latest || return

  local cid_before out
  cid_before="$(container_id)"
  out="$("${ROOT_DIR}/scripts/update.sh" latest --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "nothing to do" "reports that nothing needs doing"
  assert_eq "$(container_id)" "${cid_before}" "container was not recreated"
}

scenario_missing_image() {
  scenario "4. an unavailable target image changes nothing"
  baseline_up v2.0.0 || return

  local cid_before out
  cid_before="$(container_id)"
  out="$("${ROOT_DIR}/scripts/update.sh" v9.9.9 --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "not available locally" "refuses to continue"
  assert_eq "$(container_id)" "${cid_before}" "container untouched"
  if wait_http 20; then ok "application still serving"; else bad "application stopped serving"; fi
}

scenario_pull_failure() {
  scenario "5. a failed image pull with no local copy changes nothing"
  baseline_up v2.0.0 || return

  local cid_before out
  cid_before="$(container_id)"
  # An unreachable registry: the pull fails and the tag is not present locally.
  set_env XN_IMAGE "127.0.0.1:1/xiaowork-nav"
  out="$("${ROOT_DIR}/scripts/update.sh" v2.0.1 --yes 2>&1 || true)"
  assert_contains "${out}" "could not pull" "reports the pull failure"
  assert_eq "$(container_id)" "${cid_before}" "container untouched"
  assert_eq "$(get_env APP_VERSION)" "v2.0.0" "APP_VERSION untouched"

  write_env v2.0.0
  dump "${out}"
}

scenario_start_failure() {
  scenario "6. a container that fails to start rolls back (schema unchanged)"
  baseline_up v2.0.0 || return
  ensure_seeded || return

  write_env v2.0.0 v2.0.1 ""   # v2.0.1 fails to start; the current version stays v2.0.0
  local out
  out="$("${ROOT_DIR}/scripts/update.sh" v2.0.1 --compose "${TESTFAIL_COMPOSE}" --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "update failed" "detects the failure"
  assert_contains "${out}" "rollback verified" "verifies the rollback"
  assert_eq "$(get_env APP_VERSION)" "v2.0.0" "APP_VERSION restored"
  assert_eq "$(db_field schema_version)" "${EXPECTED_SCHEMA}" "schema unchanged"
  assert_eq "$(db_field bookmarks)" "1" "data intact"
  dump "${out}"
}

scenario_health_failure_after_migration() {
  scenario "7. health failure after a migration restores the snapshot"
  baseline_up v2.0.0 || return
  ensure_seeded || return
  local schema_before bookmarks_before
  schema_before="$(db_field schema_version)"
  bookmarks_before="$(db_field bookmarks)"

  write_env v2.0.0 "" v2.0.2   # v2.0.2 migrates, then never reports healthy
  local out
  out="$("${ROOT_DIR}/scripts/update.sh" v2.0.2 --compose "${TESTFAIL_COMPOSE}" --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "migrated the database" "detects that the schema advanced"
  assert_contains "${out}" "rollback verified" "verifies the rollback"
  assert_eq "$(get_env APP_VERSION)" "v2.0.0" "APP_VERSION restored"
  assert_eq "$(db_field schema_version)" "${schema_before}" "schema restored to the pre-update value"
  assert_eq "$(db_field bookmarks)" "${bookmarks_before}" "bookmark count restored"
  dump "${out}"
}

scenario_unknown_db_version() {
  scenario "8. a new image that cannot report the schema restores the snapshot"
  baseline_up v2.0.0 || return
  ensure_seeded || return

  write_env v2.0.0 test-broken ""   # exits immediately and cannot read the database
  local out
  out="$("${ROOT_DIR}/scripts/update.sh" test-broken --compose "${TESTFAIL_COMPOSE}" --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "cannot determine the database schema version" "takes the fail-safe path"
  assert_contains "${out}" "rollback verified" "verifies the rollback"
  assert_eq "$(get_env APP_VERSION)" "v2.0.0" "APP_VERSION restored"
  assert_eq "$(db_field schema_version)" "${EXPECTED_SCHEMA}" "schema intact"
  assert_eq "$(db_field bookmarks)" "1" "data intact"
  dump "${out}"
}

scenario_schema_readable_while_stopped() {
  scenario "9. schema version is readable while the application is stopped"
  baseline_up v2.0.0 || return

  compose stop app >/dev/null
  assert_eq "$(schema_version 2>/dev/null || true)" "${EXPECTED_SCHEMA}" "schema read from the database file with the app stopped"
  assert_not_eq "$(db_field categories 2>/dev/null || true)" "" "row counts also readable while stopped"

  compose up -d >/dev/null
  if wait_for_health 120 >/dev/null; then ok "restarted after the check"; else bad "could not restart after the check"; fi
}

scenario_restore_with_version() {
  scenario "10. restore.sh switches to the snapshot version before starting"
  baseline_up v2.0.0 || return
  ensure_seeded || return

  compose stop app >/dev/null
  local snapshot
  snapshot="$(snapshot_data_dir)"
  compose up -d >/dev/null
  wait_for_health 120 >/dev/null || { bad "could not restart"; return; }

  # Pretend a newer version is configured; the restore must move back.
  write_env v2.0.1

  local out
  out="$("${ROOT_DIR}/scripts/restore.sh" "${snapshot}" --yes 2>&1 || true)"
  assert_contains "${out}" "switching APP_VERSION" "announces the version switch"
  assert_contains "${out}" "restore complete" "restores successfully"
  assert_eq "$(get_env APP_VERSION)" "v2.0.0" "APP_VERSION follows the snapshot"
  assert_eq "$(db_field bookmarks)" "1" "data present after the restore"
}

scenario_restore_without_version() {
  scenario "11. restore.sh refuses to start when the snapshot version is unknown"
  baseline_up v2.0.0 || return

  compose stop app >/dev/null
  local snapshot renamed
  snapshot="$(snapshot_data_dir)"
  renamed="${BACKUP_DIR}/snapshot-without-version.tar.gz"
  mv "${snapshot}" "${renamed}"

  local out
  out="$("${ROOT_DIR}/scripts/restore.sh" "${renamed}" --yes 2>&1 || true)"
  assert_contains "${out}" "refusing to start" "refuses to start"
  assert_contains "${out}" "application is stopped" "leaves the application stopped"

  compose up -d >/dev/null
  if wait_for_health 120 >/dev/null; then ok "could restart afterwards"; else bad "could not restart afterwards"; fi
}

scenario_upgrade_from_release() {
  scenario "13. upgrading from the released v2.0.1 keeps data and manual icons"
  if ! docker image inspect xiaowork-nav:v2.0.1-old >/dev/null 2>&1; then
    bad "the v2.0.1 image is unavailable; cannot test the upgrade path"
    return
  fi

  # Start from an empty data directory so the old image can own the schema.
  compose down -v >/dev/null 2>&1 || true
  rm -rf "${DATA_DIR:?}"/* 2>/dev/null || true

  write_env v2.0.1-old
  compose up -d >/dev/null
  wait_for_health 120 >/dev/null || { bad "v2.0.1 baseline did not become healthy"; return; }
  assert_eq "$(schema_version 2>/dev/null || true)" "2" "released image uses the previous schema"

  curl -fsS -X POST "${base_url}/api/v1/setup" -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"deploy-test-pass"}' >/dev/null
  local cookie
  cookie="$(login_cookie)"
  [[ -n "${cookie}" ]] || { bad "could not log in"; return; }

  printf '%s' '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" fill="#7c6cff"/></svg>' \
    > "${WORK_ROOT}/manual-icon.svg"
  local uploaded icon_url
  # The host-style path keeps curl (a native binary) able to open the file on
  # Windows, where MSYS would otherwise rewrite the POSIX path.
  uploaded="$(curl -fsS -X POST "${base_url}/api/v1/admin/uploads/icon" -H "Cookie: ${cookie}" \
    -F "file=@${WORK_ROOT_HOST}/manual-icon.svg;type=image/svg+xml" 2>/dev/null || true)"
  icon_url="$(printf '%s' "${uploaded}" | sed -n 's/.*"url":"\([^"]*\)".*/\1/p')"
  assert_not_eq "${icon_url}" "" "manual SVG icon uploaded"
  if [[ -z "${icon_url}" ]]; then
    dump "${uploaded}"
    return
  fi

  curl -fsS -X POST "${base_url}/api/v1/admin/categories" -H 'Content-Type: application/json' \
    -H "Cookie: ${cookie}" -d '{"name":"升级测试"}' >/dev/null
  curl -fsS -X POST "${base_url}/api/v1/admin/bookmarks" -H 'Content-Type: application/json' \
    -H "Cookie: ${cookie}" \
    -d "{\"category_id\":1,\"title\":\"手动图标站点\",\"url\":\"https://example.com/\",\"icon_type\":\"upload\",\"icon_value\":\"${icon_url}\"}" >/dev/null
  curl -fsS -X POST "${base_url}/api/v1/admin/bookmarks" -H 'Content-Type: application/json' \
    -H "Cookie: ${cookie}" \
    -d '{"category_id":1,"title":"普通站点","url":"https://example.org/"}' >/dev/null

  local cats_before bookmarks_before
  cats_before="$(db_field categories)"
  bookmarks_before="$(db_field bookmarks)"
  assert_eq "${cats_before}" "1" "one category before the upgrade"
  assert_eq "${bookmarks_before}" "2" "two bookmarks before the upgrade"

  # APP_VERSION still points at the released image, so update.sh performs a
  # real upgrade instead of reporting a no-op.
  local out
  out="$("${ROOT_DIR}/scripts/update.sh" v2.0.1 --compose compose.yaml --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "update complete" "upgrade reports success"
  assert_eq "$(get_env APP_VERSION)" "v2.0.1" "APP_VERSION is the target"
  assert_eq "$(schema_version 2>/dev/null || true)" "${EXPECTED_SCHEMA}" "schema advanced to the new version"
  assert_eq "$(db_field categories)" "${cats_before}" "category count unchanged"
  assert_eq "$(db_field bookmarks)" "${bookmarks_before}" "bookmark count unchanged"

  local status
  status="$(curl -s -o /dev/null -w '%{http_code}' "${base_url}${icon_url}")"
  assert_eq "${status}" "200" "manual uploaded icon is still served after the upgrade"

  local uploads
  uploads="$(db_field uploads)"
  assert_eq "${uploads}" "1" "uploaded icon metadata survived the upgrade"
  dump "${out}"
}

scenario_unwritable_data_dir() {
  scenario "14. an unwritable data directory fails fast with guidance"

  local out
  out="$(MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*' docker run --rm \
    -u 12345:12345 -v "${DATA_DIR_HOST}:/data:ro" \
    xiaowork-nav:v2.0.1 --db-info --db /data/app.db 2>&1 || true)"

  assert_contains "${out}" "is not writable" "detects the unwritable directory"
  assert_contains "${out}" "chown" "prints an actionable fix"
  assert_contains "${out}" "Never use chmod 777" "explicitly warns against loosening permissions"
}

scenario_npm_mode_failure() {
  scenario "12. an NPM-style deployment (no host port) rolls back after a failed upgrade"
  docker network create "${NPM_TEST_NETWORK}" >/dev/null 2>&1 || true
  baseline_up v2.0.0 || return
  ensure_seeded || return
  local schema_before bookmarks_before
  schema_before="$(db_field schema_version)"
  bookmarks_before="$(db_field bookmarks)"

  write_env v2.0.0 "" v2.0.2 "${NPM_TEST_NETWORK}"
  local out
  out="$("${ROOT_DIR}/scripts/update.sh" v2.0.2 --compose "${NPMTEST_COMPOSE}" --no-pull --yes 2>&1 || true)"
  assert_contains "${out}" "migrated the database" "detects the migration in NPM mode"
  assert_contains "${out}" "rollback verified" "verifies the rollback in NPM mode"
  assert_eq "$(get_env APP_VERSION)" "v2.0.0" "APP_VERSION restored"
  assert_eq "$(db_field schema_version)" "${schema_before}" "schema restored"
  assert_eq "$(db_field bookmarks)" "${bookmarks_before}" "data intact"

  # The rolled-back container must be healthy without any published host port.
  local status
  status="$(container_health "$(container_id)")"
  assert_eq "${status}" "healthy" "rolled-back container healthy without a host port"
  dump "${out}"
}

# ---------- main ----------

main() {
  command -v docker >/dev/null 2>&1 || { echo "docker is required"; exit 2; }
  docker info >/dev/null 2>&1 || { echo "the Docker daemon is not reachable"; exit 2; }

  mkdir -p "${WORK_ROOT}"
  if [[ -f "${DEPLOY_DIR}/.env" ]]; then
    ENV_BACKUP="${WORK_ROOT}/env.backup"
    cp "${DEPLOY_DIR}/.env" "${ENV_BACKUP}"
  fi
  LATEST_BACKUP_ID="$(docker image inspect --format '{{.Id}}' xiaowork-nav:latest 2>/dev/null || true)"

  write_fail_compose "${DEPLOY_DIR}/${TESTFAIL_COMPOSE}"
  write_fail_compose "${DEPLOY_DIR}/${NPMTEST_COMPOSE}" "${NPM_TEST_NETWORK}"

  build_images

  scenario_db_inspection
  scenario_pinned_noop
  scenario_latest_unchanged
  scenario_missing_image
  scenario_pull_failure
  scenario_start_failure
  scenario_health_failure_after_migration
  scenario_unknown_db_version
  scenario_schema_readable_while_stopped
  scenario_restore_with_version
  scenario_restore_without_version
  scenario_npm_mode_failure
  scenario_upgrade_from_release
  scenario_unwritable_data_dir

  printf '\n'
  if [[ "${FAILED}" -eq 0 ]]; then
    printf '%sAll deployment fault-injection tests passed.%s\n' "${c_green}" "${c_reset}"
  else
    printf '%sSome deployment tests failed.%s\n' "${c_red}" "${c_reset}"
    exit 1
  fi
}

main "$@"
