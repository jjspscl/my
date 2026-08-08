# my — agent rules

`my` is a personal dashboard monorepo: Go API + React/Vite frontend + single-binary production runtime.

## Source of truth

Use these files in this order:

1. This `AGENTS.md` for workflow, safety, and current-state guidance.
2. `docs/opencode.md` for OpenCode-specific config, agents, and MCP notes.
3. `docs/architecture.md`, `docs/backend-ddd.md`, `docs/frontend.md`, and `docs/offline-sync.md` for subsystem details.
4. `ROADMAP.md` for aspirational/future work only — do not treat it as current implementation truth.
5. `HANDOFF.md` is ephemeral task handoff state, not durable architecture documentation.

Read deeper docs lazily when relevant. Do not preload all of them unless task scope needs it.

## Canonical commands

Run repo-level commands from the project root:

- `mise run dev` — start Redis, Mailpit, API, and Vite dev servers
- `mise run build` — build Vite assets, copy them into Go embed dir, build `bin/my`
- `mise run test` — run Go and frontend tests
- `mise run lint` — run `go vet` and frontend ESLint
- `mise run typecheck` — run frontend TypeScript checks
- `mise run migrate` — run API migrations
- `mise run seed` — seed local dev data
- `mise run clean` — remove build outputs

## Runtime facts

### Production

- `my` ships as one Go binary.
- Frontend assets are built by Vite, copied into `apps/api/internal/platform/web/static/`, and embedded with `go:embed`.
- Go serves both `/api/v1/*` and the SPA.

### Development

- Go API: `http://localhost:8080`
- Vite dev server: `http://localhost:5173`
- Vite proxies `/api` to the Go API.
- `mise run dev` also starts:
  - Redis on `:6379`
  - Mailpit SMTP on `:1025`
  - Mailpit UI on `:8025`

## Current architecture

### Backend

Current implemented slices live under `apps/api/internal/contexts/`:

- `access` — magic-link auth, session lifecycle
- `finance` — transactions, budgets, bills, goals, wallets, transfers
- `habits` — habits, completions, streaks

Scaffold-only placeholder contexts also exist (`dashboard`, `health`, `identity`, `modules`, `notifications`, `sync`) but are not full vertical slices yet.

Platform code lives under `apps/api/internal/platform/`.
Shared middleware/response helpers live under `apps/api/internal/shared/`.

### Frontend

Feature code lives under `apps/web/src/features/`.
Shared sync/network infra lives under `apps/web/src/shared/sync/`.
Shared UI primitives live under `apps/web/src/components/ui/`.

Frontend conventions:

- infer app data types from Zod schemas
- use TanStack Query for server state
- use TanStack Router for route + URL state
- use Zustand only for ephemeral local state such as network/sync status

## Auth truth

Auth is single-user magic-link auth, not JWT/refresh-token auth.

- request magic link via `/api/v1/auth/magic-link`
- verify via `/api/v1/auth/verify`
- authenticated session stored in Redis-backed opaque session cookie `my_session`
- CSRF token stored in JS-readable cookie `my_csrf` and sent as `X-CSRF-Token`

## Offline/sync truth

Offline infrastructure exists but is partial.

Current state:

- IndexedDB mutation queue via `idb-keyval`
- queue schema validated with Zod
- online/offline state tracked with Zustand
- drain runs on startup, on reconnect, and every 30s
- replay re-sends original HTTP mutations with cookies + CSRF header
- Vite PWA uses Workbox `NetworkFirst` runtime caching for `/api/v1/*`

Important limitation:

- there is no dedicated `/api/v1/sync/*` backend yet
- `offlineMutate()` infra exists, but not all feature mutations use it yet
- do not document offline sync as complete end-to-end behavior

## OpenCode workflow

Project uses OpenCode with repo-local config in `opencode.jsonc` and repo-local skills in `.opencode/skills/`.

Default workflow:

- `plan` is default agent
- `plan` scopes work, prepares/reviews `HANDOFF.md`, and verifies output
- `build` executes implementation work
- `HANDOFF.md` is temporary coordination state for the current task only
- before any commit, push, or PR task, load repo-local `git-commit-and-push` skill

## MCP/tooling rules

Current enabled MCPs in OpenCode config:

- `brave-search` — web search
- `context7` — library docs
- `filesystem` — repo-scoped file access
- `playwright` — browser automation for PWA/offline/browser flows
- `gh_grep` — GitHub code example search
- `engram` — local persistent memory MCP, project-scoped to `my`

Guidance:

- use `context7` for library/framework docs first
- use `brave-search` for general web/current-state research
- use `playwright` only when browser-state or PWA verification is required
- use `gh_grep` when public code examples are more useful than prose docs
- use `mem_context` after compaction or context reset; save significant decisions and discoveries with `mem_save`
- keep tokens and PATs out of committed files

## Documentation rules

When code or runtime behavior changes:

- update `AGENTS.md` if workflow, commands, or architectural truth changed
- update `README.md` for user-facing setup/runtime changes
- update subsystem docs when implementation reality changes
- keep current-state docs separate from roadmap/speculative work

## Safety

- never commit secrets, `.env`, PATs, or auth tokens
- keep finance/personal-data tooling conservative by default
- prefer minimal, reversible config changes
- document assumptions when behavior is partial or scaffold-only
