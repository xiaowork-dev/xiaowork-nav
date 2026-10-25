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
  script with automatic pre-update snapshot and safe rollback.

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
./scripts/update.sh v2.0.1      # or: ./scripts/update.sh latest
```

The update script:

1. records the current version and database schema version,
2. stops the app and takes a **consistent snapshot** of the data directory,
3. switches the version atomically and starts the new image,
4. verifies health and the schema version,
5. on failure restores the snapshot when the new build migrated the database,
   then starts the previous image again.

Your database, uploaded icons and settings are never overwritten. Manual
backup/restore is also available:

```bash
./scripts/backup.sh
./scripts/restore.sh /path/to/backup-2.0.0-20261025-101500.tar.gz
```

See [docs/updating.md](docs/updating.md) for rollback and migration
compatibility.

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
