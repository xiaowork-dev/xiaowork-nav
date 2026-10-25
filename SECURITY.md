# Security Policy

## Supported versions

Security fixes are applied to the latest released minor version.

| Version | Supported |
| --- | --- |
| 2.0.x | ✅ |
| < 2.0 | ❌ |

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Report privately via GitHub Security Advisories
("Security" → "Report a vulnerability") on
`https://github.com/xiaowork-dev/xiaowork-nav`, or by email to the maintainers
listed in the repository profile.

Please include:

- the affected version (`/api/v1/version`),
- a description of the issue and its impact,
- steps to reproduce (a minimal request or configuration is ideal),
- any suggested fix.

We aim to acknowledge reports within a few days. Please allow time for a fix
before public disclosure.

## Deployment hardening checklist

XiaoWork Nav is designed to run behind a TLS-terminating reverse proxy.

- [ ] Terminate HTTPS at the reverse proxy; never expose the app port directly to
      the internet.
- [ ] Keep `XN_BIND_ADDR=127.0.0.1` (default) or use the NPM mode that publishes
      no host port at all.
- [ ] Keep `XN_SECURE_COOKIE=true` when serving over HTTPS (set `false` only for
      local HTTP debugging).
- [ ] Set `XN_TRUSTED_PROXIES` to your proxy's CIDR so `X-Forwarded-*` headers are
      only honoured from a trusted hop.
- [ ] Use a long, unique administrator password; the first-run wizard is the only
      way to create the account and closes afterwards.
- [ ] Back up `/data` regularly and test a restore
      (see [docs/updating.md](docs/updating.md)).
- [ ] Do **not** mount `/var/run/docker.sock` into the container.

## Built-in protections

| Area | Measure |
| --- | --- |
| Passwords | Argon2id (PHC string format, per-password random salt) |
| Sessions | Server-side sessions, `HttpOnly` + `SameSite=Lax` cookie, `Secure` in production, sliding expiry, invalidated on password change |
| CSRF | `Origin`/`Referer` check on state-changing requests |
| Login abuse | Rate limiting per IP **and** per username |
| URLs | Only `http`/`https` accepted; `javascript:`/`data:`/`file:` rejected |
| Icon fetching | SSRF protection: resolved addresses are checked against loopback, private, link-local and cloud-metadata ranges; redirects are limited |
| Uploads | Size cap (1 MiB), MIME allow-list, extension/filename allow-list, no path traversal |
| SVG | Served as `image/svg+xml` with `X-Content-Type-Options: nosniff` and a sandboxing `Content-Security-Policy`; referenced through `<img>`, never inlined as HTML |
| Data exposure | `/data` is never served statically; `.env` and databases are excluded from git |
| Container | Non-root user, `no-new-privileges`, no host port in proxy mode |
| Downgrade | The service refuses to start against a database migrated by a newer build |

## Secrets

The repository must never contain real credentials, tokens, private keys, user
databases, exported navigation data or screenshots rendered from real data.
CI enforces this with a repository-safety job; `.gitignore` excludes `.env`,
`data.json`, `*.db` and `docs/screenshots/`.
