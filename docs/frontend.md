# Frontend

## Stack

- React 19
- Vite 8
- TypeScript 6 (strict)
- TanStack Router
- TanStack Query
- Zod 4
- React Hook Form + `zodResolver`
- Zustand for small local client state only
- Tailwind CSS v4
- shadcn/ui-style component set
- `vite-plugin-pwa`

## Rules

1. Infer application data types from Zod schemas.
2. Do not cast `response.json()` directly to application types.
3. Parse API responses with schemas.
4. Keep server state in TanStack Query.
5. Keep route/search state in TanStack Router.
6. Use Zustand only for ephemeral client state such as network/sync status.
7. Keep feature-local query key factories next to the feature API layer.

## Directory structure

```text
src/
  components/      shared UI, layout, widgets
  features/        feature modules
  routes/          TanStack Router route tree
  shared/          API client, sync infra, utilities
```

Typical feature layout:

```text
features/<name>/
  api/
  components/
  hooks/
  lib/
  schemas/
```

## Current offline/state note

The repo currently uses Zustand inside `src/shared/sync/` for:

- network connectivity state
- sync queue status

This is acceptable because it is ephemeral local client state, not server state.

## Scripts

### Preferred repo-level entrypoints

Run from repo root with `mise`:

| Command | Purpose |
|---|---|
| `mise run dev` | full dev stack |
| `mise run build` | frontend build + embed copy + Go build |
| `mise run test` | API + frontend tests |
| `mise run lint` | API vet + frontend lint |
| `mise run typecheck` | frontend typecheck |

### App-local scripts

Inside `apps/web/`:

| Script | Purpose |
|---|---|
| `pnpm dev` | Vite dev server with HMR |
| `pnpm build` | TypeScript build + Vite build |
| `pnpm typecheck` | `tsc --noEmit` |
| `pnpm lint` | ESLint |
| `pnpm test` | Vitest |
| `pnpm route:generate` | TanStack Router code generation |
| `pnpm e2e` | Playwright tests |

### E2E runtime requirements

The Playwright specs drive the real magic-link flow and run against the
**production binary** in binary mode: the suite's `webServer` starts
`bin/my` (build it with `mise run build`) and tests the SPA it serves at
`http://localhost:8080`.

```bash
mise run build
MY_USER_EMAIL=you@example.com MY_WEB_URL=http://localhost:8080 \
  MY_MAGIC_LINK_RATE=100 pnpm --filter @my/web e2e
```

Environment variables (defaults in parentheses):

- `E2E_EMAIL` (`jjspscl@gmail.com`) — login email; **must equal
  `MY_USER_EMAIL`** or the backend silently no-ops and the suite times out
- `E2E_BASE_URL` (`http://localhost:5173`) — origin the suite targets;
  CI uses `http://localhost:8080`
- `E2E_MAILPIT_URL` (`http://localhost:8025/api/v1`) — Mailpit HTTP API
- `MY_MAGIC_LINK_RATE=100` — required: the suite issues 7 magic-link
  requests; the production default (6 per 15 min per IP) trips a 429
  mid-run

The CI workflow (`.github/workflows/e2e-ci.yml`) starts Redis and
Mailpit as service containers, builds the binary, and uploads the
Playwright report on failure. It is advisory until it proves stable,
then promoted to a required check.
