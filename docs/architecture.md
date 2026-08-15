# Architecture

## Overview

`my` is a personal dashboard built as a Go backend plus React/Vite frontend.

Production ships as one Go binary with embedded frontend assets.
Development runs the API and Vite separately.

## Production runtime

```text
┌─────────────────────────────┐
│        Go binary (my)       │
├─────────────────────────────┤
│ chi router                  │
│ ├── /api/v1/*               │
│ └── /* -> embedded SPA      │
├─────────────────────────────┤
│ embedded static assets      │
│ (Vite build output)         │
├─────────────────────────────┤
│ libSQL / SQLite             │
│ Redis                       │
└─────────────────────────────┘
```

Build flow:

1. Vite builds to `apps/web/dist`
2. `scripts/copy-web-assets.sh` copies assets into `apps/api/internal/platform/web/static/`
3. Go embeds that directory with `go:embed`
4. `go build` produces `bin/my`

## Development runtime

```text
┌──────────────┐     ┌──────────────┐
│ Vite :5173   │---->│ Go API :8080 │
│ React HMR    │proxy│ chi router   │
│ /api -> 8080 │     │ libSQL+Redis │
└──────────────┘     └──────────────┘
```

`mise run dev` also brings up Redis and Mailpit.

## Backend layout

Backend code lives under `apps/api/internal/`.

### Active vertical slices

Current implemented slices under `apps/api/internal/contexts/`:

- `access`
- `finance`
- `habits`
- `intelligence` — confidence-gated LLM analysis (suggestions only; see
  `docs/intelligence.md`)

### Scaffold-only contexts

These folders exist as placeholders but are not full production slices yet:

- `dashboard`
- `health`
- `identity`
- `modules`
- `notifications`
- `sync`

### Slice structure

```text
apps/api/internal/contexts/<name>/
  domain/          entities, value objects, repository ports
  application/     application services/use cases
  infrastructure/  adapters and repository implementations
  interfaces/http/ HTTP handlers and route wiring
```

Not every scaffolded context currently has every layer populated.

### Platform and shared code

```text
apps/api/internal/platform/    bootstrap, config, database, logger, mcp, redis, session, update, version, web
apps/api/internal/shared/      middleware, response helpers
```

`internal/platform/mcp` is an inbound transport adapter across active contexts,
not a bounded context. It exposes application services through stdio and
optional streamable HTTP without making HTTP self-calls. HTTP MCP uses a
dedicated listener so bind policy is enforced by the OS, not request headers.

## Frontend layout

Frontend code lives under `apps/web/src/`.

```text
src/
  components/      shared UI, layout, widgets
  features/        feature modules
  routes/          TanStack Router route tree
  shared/          API helpers, sync infra, utilities
```

Feature modules generally follow:

```text
features/<name>/
  api/
  components/
  hooks/
  lib/
  schemas/
```

## Key principles

- domain logic stays outside transport/framework concerns
- handlers stay thin: validate -> delegate -> map response
- frontend app data types come from Zod schemas
- TanStack Query owns server state
- TanStack Router owns route/URL state
- Zustand is reserved for local ephemeral client state such as network/sync state
