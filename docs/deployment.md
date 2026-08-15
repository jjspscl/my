# Deployment

## Production binary

```bash
mise run build
./bin/my
```

The production binary serves both API and frontend on a single port (default `8080`).

## Build flow

`mise run build` does three things:

1. builds the frontend in `apps/web/dist`
2. copies those assets into `apps/api/internal/platform/web/static/`
3. builds the Go binary at `bin/my`

## Environment variables

See `.env.example` for the current supported environment variables.

Key runtime categories:

- API/web ports and URLs
- libSQL/SQLite database settings
- optional Turso remote settings
- Redis session store
- magic-link auth settings
- SMTP settings
- default currency
- MCP-related search/doc API keys
- intelligence/LLM settings: `MY_LLM_ENABLED`, `MY_LLM_MASTER_KEY`,
  `MY_LLM_CODEX_PATH` (see `docs/intelligence.md`)

## Docker

A production image is maintained at `infrastructure/docker/Dockerfile` and
published to `ghcr.io/jjspscl/my` on `v*` tags (`.github/workflows/docker.yml`).

```bash
docker compose up -d --build          # full stack: api + redis
MY_USER_EMAIL=you@example.com docker compose up -d   # required env
```

### Image contract

- **Runtime**: `gcr.io/distroless/static:nonroot` — no shell, no curl,
  runs as UID 65532. Probe it externally (HTTP `/api/v1/health`); there is
  no in-image `HEALTHCHECK`.
- **Self-contained**: frontend assets, SQL migrations, and tzdata are all
  embedded in the binary. Nothing else is shipped alongside it.
- **Database**: `MY_DATABASE_URL=file:/data/my.db`; `/data` is a volume
  seeded writable for the nonroot user. SQLite WAL/journal files land
  alongside the DB, so mount a directory, not a file.
- **Migrations**: applied automatically at boot from the embedded set.
  A fresh boot builds the full schema; a broken or empty migration set is
  a hard error, never a silent empty schema.
- **Shutdown**: the binary is PID 1 and drains gracefully on SIGTERM
  (~15s); `stop_grace_period: 20s` in compose covers it.
- **Redis**: mandatory config, but the client is lazy — an unreachable
  Redis does not block boot and only fails the requests that need it.
- **MCP**: stays disabled in the container (`MY_MCP_ENABLED=false`). Its
  default loopback bind is unreachable from outside a container, and
  widening it exposes the full finance surface behind only a static token.
- **Required env**: `MY_USER_EMAIL` (no default; boot fails without it).

### Build notes

- Multi-arch (`linux/amd64` + `linux/arm64`) via buildx; `-p 2` bounds
  Go compile memory so the image builds on hosts with ~2GB of Docker
  memory (CI runners are unaffected).
- `rediss://` (TLS Redis) URLs are currently rejected at boot — managed
  Redis must use `redis://` or a sidecar.

## Backup and restore

The database is a single SQLite file in the `/data` volume. **Never back it
up by copying the file** — with WAL mode enabled a naive copy can be torn.
Use the snapshot path below (VACUUM INTO, safe on a live database).

### Taking a snapshot

```bash
mise run backup                      # -> ~/.local/share/my/backups/my-<timestamp>.db
# or, against a running container:
curl -fsS -b cookies.txt -o backup.db http://host:8080/api/v1/backup   # session-authenticated
```

JSON export (inspection, migration — not for restore): `GET /api/v1/export`
(session-authenticated) dumps all user tables, excluding `magic_tokens`.

### Restoring

1. `docker compose stop api`
2. Replace the database: `docker run --rm -v <project>_my-data:/data -v $PWD:/src alpine sh -c 'cp /src/backup.db /data/my.db'` (or `docker cp` the file into the container's `/data`)
3. `docker compose start api` — boot applies migrations idempotently and
   verifies the schema; keep the pre-restore snapshot until the app boots
   clean and data looks right.

### WAL caveats

- WAL needs a **writable database directory** (satisfied by the `/data`
  volume) and is **unsafe on network filesystems** (NFS/SMB) — use a local
  disk or block storage.
- Snapshots land in `~/.local/share/my/backups/` (outside the git working
  tree) by default.

## Probes

- `/api/v1/health` — process liveness only (always 200 while the binary
  runs). The Playwright e2e `webServer` waits on it.
- `/api/v1/ready` — dependency readiness: pings SQLite and Redis, returns
  200 only when both answer, else 503 with per-dependency status
  (`{"db":"ok","redis":"error"}`). Point orchestrator healthchecks here.

## SMTP-down escape hatch

If the mail relay is unreachable, log in from a shell:

```bash
docker compose exec api /my -login-link   # prints http://<MY_WEB_URL>/auth/verify?token=...
```

The token is single-use, 15 minutes, printed only to stdout.

## Targets

Plausible deployment targets for the single-binary runtime include:

- Docker
- Kubernetes
- Fly.io
- Railway
- Render
- AWS ECS
- GCP Cloud Run

Treat this list as deployment options, not evidence of committed production manifests.

## Database/runtime notes

- **Development DB**: embedded local file (`file:my_dev.db`)
- **Production DB**: embedded SQLite/libSQL or remote Turso/libSQL, depending on env config
- **SQLite pragmas** (local files): WAL journal mode, 5s busy timeout,
  foreign keys ON; the connection pool is bounded to a single writer
- **Redis**: session storage (sliding expiry — active sessions renew)
- **Rate limiting**: `POST /auth/magic-link` is limited per IP with an
  in-memory sliding window (`MY_MAGIC_LINK_RATE` per 15 min, default 6;
  429 + `Retry-After`). Other endpoints are unlimited.
- **Session security**: `MY_WEB_URL` must be the public origin (the
  magic-link email uses it; compose refuses to start without it). Cookies
  are Secure only over https — the binary warns at boot if it is serving
  insecure cookies on a non-local origin.
