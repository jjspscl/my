# my

A personal dashboard for finance, habits, and daily life.

> Your day, money, and habits in one calm dashboard.

## Stack

- **Backend**: Go 1.26, chi, libSQL/SQLite, Redis, slog JSON logging
- **Frontend**: React 19, Vite 8, TypeScript 6, TanStack Router, TanStack Query, Zod, React Hook Form, shadcn/ui-style components, Tailwind CSS v4
- **Architecture**: DDD-inspired modular monolith, feature-based frontend, single-binary production deploy
- **Offline**: PWA manifest + Workbox runtime caching, IndexedDB mutation queue, reconnect/interval replay engine
- **AI workflow**: OpenCode with root `AGENTS.md`, repo-local skills, optional MCP clients, and released `my-mcp` server

## Quick start

```bash
# Install pinned tools
mise install

# Install workspace dependencies
mise run install

# Start Redis, Mailpit, Go API, and Vite
mise run dev
```

## Commands

| Command | Description |
|---|---|
| `mise run install` | Install workspace dependencies |
| `mise run dev` | Start Redis, Mailpit, API, and client dev servers |
| `mise run dev:api` | Run Go API dev server (port 8080) |
| `mise run dev:client` | Run Vite client dev server (port 5173) |
| `mise run dev:redis` | Start Redis if not already running |
| `mise run dev:mail` | Start Mailpit (SMTP :1025, UI :8025) |
| `mise run dev:mail:ui` | Open Mailpit UI in browser |
| `mise run build` | Build Vite assets, copy them into Go embed dir, build production binary |
| `mise run build:mcp` | Build standalone MCP stdio server |
| `mise run test` | Run all tests |
| `mise run lint` | Run Go vet and frontend ESLint |
| `mise run typecheck` | Run frontend type checking |
| `mise run migrate` | Run database migrations |
| `mise run seed` | Seed local dev data |
| `mise run clean` | Clean build outputs |
| `mise run release:guard` | Pre-flight release checks (pass version, optional `--fix`) |
| `mise run release:check` | Snapshot GoReleaser build without publishing |
| `MY_RELEASE_VERSION=v1.0.0 mise run release:tag` | Validate and create annotated release tag; push manually |

## Runtime

### Production

`my` runs as a single Go binary:

1. Vite builds frontend assets.
2. `scripts/copy-web-assets.sh` copies them into `apps/api/internal/platform/web/static/`.
3. Go embeds the static assets with `go:embed`.
4. One Go server handles both `/api/v1/*` and SPA routes.

### Development

Two servers run independently:

- **Go API** on `http://localhost:8080`
- **Vite** on `http://localhost:5173`

Vite proxies `/api` to the Go API.

### Dev services

`mise run dev` also ensures these containers are available:

- **Redis** (`my-redis`) — session storage, port 6379
- **Mailpit** (`my-mailpit`) — local SMTP server, port 1025
- **Mailpit UI** — `http://localhost:8025`

## Current architecture

```text
my/
  AGENTS.md                         Project agent/source-of-truth rules
  apps/api/                         Go API, migrations, embedded web runtime
  apps/web/                         React/Vite frontend
  docs/                             Current-state architecture and workflow docs
  .opencode/                        Repo-local OpenCode skills
  skills/                           End-user finance agent skills (runtime)
  agent/                            Finance agent profile (SOUL + wiring)
  infrastructure/                   Infra placeholders
  deployments/                      Deployment placeholders
  packages/                         Shared package workspace (currently light use)
```

### Backend slices in use

Active backend slices live in `apps/api/internal/contexts/`:

- `access`
- `finance`
- `habits`

Other context folders exist as scaffolds, but are not full vertical slices yet.

### Frontend shape

- `src/features/*` — feature modules
- `src/components/ui/*` — shared UI primitives
- `src/shared/sync/*` — offline/network/sync infrastructure

## Auth and offline status

### Auth

Current auth is **single-user magic-link auth**.

- Redis-backed opaque session cookie: `my_session`
- JS-readable CSRF cookie: `my_csrf`
- no JWT access tokens or refresh token rotation in the current implementation
- cookie `Secure` behavior follows `MY_WEB_URL` by default; set `MY_SECURE_COOKIES` explicitly when needed

### Offline/sync

Current offline support is **partial but real**.

Implemented:

- PWA manifest and service worker
- Workbox `NetworkFirst` runtime caching for `/api/v1/*`
- IndexedDB mutation queue
- reconnect + periodic queue draining

Not implemented yet:

- dedicated `/api/v1/sync/*` API
- universal offline queue adoption across all feature mutations

## OpenCode / agent workflow

Root agent guidance lives in `AGENTS.md`.

Repo-local OpenCode config lives in `opencode.jsonc`.

Current workflow:

- `plan` is the default OpenCode agent
- `build` executes implementation work
- repo-local skills live in `.opencode/skills/`
- end-user finance agent skills live in `skills/`; profile in `agent/finance/`

### MCP server for coding agents

Install latest standalone MCP server on macOS or Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/jjspscl/my/v1.3.0/scripts/install-mcp.sh | sh
```

Installer verifies release checksum and GitHub attestation when `gh` is available. It prints client configuration; it never edits agent config files.

`my-mcp` uses stdio and needs direct access to the same database and Redis as the dashboard, so it only works on a machine that already holds your `my` data. Configure it in Codex, Claude Code, opencode, Hermes, or OpenClaw using `~/.local/bin/my-mcp`.

For HTTP access, run the dashboard with `MY_MCP_ENABLED=true` and a 32+ character `MY_MCP_TOKEN`. MCP listens on its own loopback-bound listener at `http://127.0.0.1:8081/mcp`, separate from the dashboard port.

Full tool/resource/prompt inventory, client configuration, security model, release, and update instructions: [`docs/mcp.md`](docs/mcp.md).

Agents can check installed binary version without self-updating:

```bash
my-mcp --check-update
```

### MCP/tooling

Enabled OpenCode MCPs:

- `brave-search`
- `context7`
- `filesystem`
- `playwright`
- `gh_grep`
- `engram`

Use heavier MCPs only when the task actually needs browser automation or public code example lookup.

## Docs index

- `AGENTS.md` — project rules and current-state truth for agents
- `docs/opencode.md` — OpenCode config, agents, MCP notes
- `docs/mcp.md` — provided MCP server, install, clients, and releases
- `docs/architecture.md` — runtime and directory architecture
- `docs/backend-ddd.md` — backend patterns and current auth/data reality
- `docs/frontend.md` — frontend conventions
- `docs/offline-sync.md` — current offline/sync implementation status
- `ROADMAP.md` — future work, not current implementation truth

## License

MIT
