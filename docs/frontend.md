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

The Playwright specs drive the real magic-link flow (Mailpit at
`localhost:8025`). Each `beforeEach` requests a link and `retries: 1`
doubles the count, so run the API with a raised limit:

```bash
MY_MAGIC_LINK_RATE=100 mise run dev:api
```

Without this, the default limit (6 per 15 min per IP) trips a 429
mid-suite and flakes the run.
