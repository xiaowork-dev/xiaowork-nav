# XiaoWork Nav

A lightweight, self-hosted personal navigation start page — a searchable grid of
your bookmarks, grouped by category, with a full admin UI and one-command
Docker deployment.

Built as a single container: **Go** serves both the REST API and the Vue 3
frontend, with **SQLite** for storage. No external database, no Node runtime in
production.

```
┌──────────────┐   HTTPS   ┌────────────────┐   HTTP    ┌─────────────────────────┐
│   Browser    │ ────────▶ │ Reverse proxy  │ ────────▶ │  xiaowork-nav container │
└──────────────┘           │ (NPM / Nginx)  │           │  Go API + Vue SPA       │
                           └────────────────┘           │        └── SQLite /data │
                                                        └─────────────────────────┘
```

## Features

- **Navigation home page** — category groups, icon tiles, clock, live search and
  a switchable search engine. Responsive down to 390px.
- **Two navigation layouts** — *scroll* (every category stacked, with a fixed
  category rail on desktop) and *tabs* (one category at a time behind a
  horizontal tab strip). Switch with one click; the choice is remembered
  locally and is independent of the light/dark theme.
- **Automatic site icons** — for sites without a manually chosen icon, the
  server fetches the icon the site itself declares (web app manifest,
  apple-touch-icon, `rel=icon`), falls back to `/favicon.ico`, and caches it in
  `/data/uploads`. A clearly labelled brand guess is the last resort. Icons you
  set yourself are never overwritten, and the admin UI can fill in everything
  missing in one background job.
- **Admin UI** — categories, bookmarks, drag-and-drop ordering, icons, site
  settings and data management, all behind a session login.
- **Icons** — external URL, local upload, verified [Simple Icons](https://simpleicons.org/)
  slug, text glyph, or a per-site favicon fallback; the first letter is shown
  while an image loads, so tiles are never blank.
- **Search** — server-side search over names, descriptions and keywords.
- **Visibility** — mark categories and bookmarks private; visitors only ever see
  public entries.
- **Data** — JSON export/import (including legacy data), online backups and
  restore.
- **Themes** — light and dark, following the system by default.
- **Deployment** — one container, `docker compose`, persistent `/data`, upgrade
  script with automatic pre-update snapshot and safe rollback. The container
  never runs as root, and the install script takes care of the data-directory
  ownership so first deployments do not hit `permission denied`.

## Quick start

Requirements: a Linux host with Docker Engine and the Docker Compose plugin.

```bash
git clone https://github.com/xiaowork-dev/xiaowork-nav.git
cd xiaowork-nav
./scripts/install.sh
```

The script validates the configuration, pulls the published image, starts the
container and waits for the health check. It prints the data directory and the
next steps when it finishes.

Then open the site and complete the **first-run wizard** to create the
administrator account.

> The default configuration binds to `127.0.0.1:8787`, so the app is not exposed
> to the internet. Put a reverse proxy in front of it (see below) or use an SSH
> tunnel: `ssh -L 8787:127.0.0.1:8787 you@server`.

### Behind Nginx Proxy Manager

```bash
cd deploy
cp .env.example .env
# set XN_PROXY_NETWORK to the Docker network your NPM container uses
../scripts/install.sh --compose compose.npm.yaml
```

This variant publishes **no host port**; NPM forwards to `xiaowork-nav:8080`
over the shared network. See [docs/installation.md](docs/installation.md).

## Updating

```bash
./scripts/update.sh v2.1.0      # or: ./scripts/update.sh latest
```

The update script:

1. checks the target tag — a pinned version that is already deployed is a no-op,
   while a mutable tag (`latest`, `v2.0`) always checks the registry and pulls the
   newest image (an unchanged image leaves the container alone),
2. reads the current version and database schema version **from the database
   file**, so it works in every Compose mode (including NPM, which publishes no
   host port) and even while the app is stopped,
3. stops the app and takes a **consistent snapshot** of the data directory,
4. switches the version atomically and starts the new image,
5. verifies health, the schema version and the bookmark count,
6. on failure rolls back and **re-verifies** the previous container; when the new
   build migrated the database (or the schema cannot be determined) the
   pre-update snapshot is restored before the previous image is started.

Your database, uploaded icons and settings are never overwritten. Manual
backup/restore is also available:

```bash
./scripts/backup.sh
./scripts/restore.sh /path/to/backup-v2.0.0-20261025-101500.tar.gz
./scripts/restore.sh <snapshot> --version v2.0.0   # start the matching version
```

`restore.sh` switches `APP_VERSION` to the snapshot's version **before** starting,
so a restored older database is never re-migrated by a newer image. When the
version cannot be determined it restores the data, leaves the app stopped and
prints instructions instead of guessing.

See [docs/updating.md](docs/updating.md) for rollback and migration
compatibility. The failure paths are covered by
[`scripts/tests/deploy-faults.test.sh`](scripts/tests/deploy-faults.test.sh).

## Configuration

All deployment settings are environment variables — nothing is hard-coded.
Copy [`deploy/.env.example`](deploy/.env.example) and edit it.

| Variable | Default | Purpose |
| --- | --- | --- |
| `XN_IMAGE` | `ghcr.io/xiaowork-dev/xiaowork-nav` | Image repository |
| `APP_VERSION` | `latest` | Image tag to run (pin a version for reproducible deploys) |
| `XN_CONTAINER_NAME` | `xiaowork-nav` | Container name / NPM forward host |
| `XN_DATA_HOST_DIR` | `./data` | Host directory mounted at `/data` |
| `XN_BIND_ADDR` | `127.0.0.1` | Standalone host bind address |
| `XN_PUBLISH_PORT` | `8787` | Standalone host port |
| `XN_PROXY_NETWORK` | – | Existing external Docker network (NPM mode) |
| `XN_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `XN_ALLOW_GUEST_ACCESS` | `true` | Allow anonymous browsing of public entries |
| `XN_SECURE_COOKIE` | `true` | Mark the session cookie `Secure` (keep `true` behind HTTPS) |
| `XN_BACKUP_KEEP` | `5` | In-app snapshots to retain |
| `XN_UPDATE_REPO` | – | `owner/name` used by the admin update check |
| `XN_TRUSTED_PROXIES` | – | CIDRs whose `X-Forwarded-*` headers are trusted |
| `XN_ALLOWED_ORIGINS` | – | Extra allowed CORS origins (development only) |
| `TZ` | `UTC` | Timezone for log timestamps |

## Development

Backend and frontend run separately; the frontend talks to the API over HTTP
only.

```bash
# backend  → http://localhost:8080
cd backend && go run ./cmd/server

# frontend → http://localhost:5173 (proxies /api to :8080)
cd frontend && npm install && npm run dev
```

Tests and checks:

```bash
cd backend  && go vet ./... && go test ./...
cd frontend && npm run build
```

Local image build:

```bash
docker build -t xiaowork-nav:local .
./scripts/deploy.sh --build
```

## Documentation

| Document | Contents |
| --- | --- |
| [docs/installation.md](docs/installation.md) | Install, first run, reverse proxy, troubleshooting |
| [docs/updating.md](docs/updating.md) | Update, backup, restore, rollback, migrations |
| [docs/architecture.md](docs/architecture.md) | Architecture and module boundaries |
| [docs/api-spec.md](docs/api-spec.md) | REST API conventions |
| [docs/openapi/openapi.yaml](docs/openapi/openapi.yaml) | OpenAPI 3.1 contract |
| [docs/database-design.md](docs/database-design.md) | Schema and migration strategy |
| [docs/frontend-agent-guide.md](docs/frontend-agent-guide.md) | Guide for replacing the frontend |
| [docs/PROGRESS.md](docs/PROGRESS.md) | Project status and delivery reports |
| [SECURITY.md](SECURITY.md) | Security policy |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) (the delivery
standard this repository follows).

## License

[MIT](LICENSE)
