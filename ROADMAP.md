# Roadmap

## Project Goals

- Personal open-source dashboard for finance and habit tracking
- Mobile-first, offline-capable PWA
- Single Go binary production deployment
- Minimal monochrome UI with shadcn/ui
- DDD modular monolith backend
- Zod schema-first frontend contracts

## MVP Checklist

- [x] Project scaffold (monorepo, mise, Go module, Vite app)
- [x] Go single-binary production runtime with embedded frontend
- [x] Vite HMR development runtime with API proxy
- [x] shadcn/ui component library (30+ components)
- [x] TanStack Router file-based routing
- [x] TanStack Query client setup
- [x] PWA manifest + service worker (via vite-plugin-pwa)
- [x] OpenCode local skills (9 skills)
- [x] mise task runner (all required commands)
- [x] Auth/session shell (magic link, login, logout, verify)
- [x] Dashboard widget registry
- [x] Finance quick expense
- [x] Monthly spend widget
- [x] Habit check-in
- [x] Today habits widget
- [x] Offline mutation queue (IndexedDB, dead-letter state)
- [x] Sync status indicator (offline badge, failed-items panel)

## Architecture Checklist

- [x] Go API with chi router
- [x] SPA fallback handler with go:embed
- [x] DDD bounded context directory structure (9 contexts)
- [x] Frontend feature-based directory structure
- [x] libSQL database integration (embedded mode via modernc.org/sqlite)
- [x] SQL migrations
- [x] Repository pattern (manual SQL)
- [x] Session cookies + CSRF double-submit
- [x] Request middleware (ID, panic recovery, request logging)
- [x] Rate limiting (magic-link endpoint)
- [x] Structured JSON logging (slog, JSON handler)
- [ ] Domain events infrastructure

## Frontend Checklist

- [x] React + Vite + TypeScript strict
- [x] TanStack Router (file-based, typed routes)
- [x] TanStack Query (configured client)
- [x] Zod (installed, ready for schemas)
- [x] shadcn/ui new-york style (30+ components)
- [x] Tailwind CSS v4 with CSS variables
- [x] PWA plugin configured
- [x] ESLint + Vitest configured
- [x] Zod schema contracts (finance, habits, dashboard, auth)
- [x] Query key factories per feature
- [x] Route loaders with ensureQueryData
- [x] Zustand ephemeral UI stores
- [x] Offline IndexedDB mutation queue (persistent, FIFO, retry + dead-letter)
- [x] React Hook Form + zodResolver forms

## Backend Checklist

- [x] Go module (github.com/jjspscl/my)
- [x] chi router with middleware
- [x] Health endpoint (/api/v1/health)
- [x] Static frontend embed + SPA handler
- [x] libSQL connection (embedded mode)
- [x] Config from environment
- [x] Structured logger (slog)
- [x] Auth context (identity, access)
- [x] Finance context (domain, application, infra, HTTP)
- [x] Habits context (domain, application, infra, HTTP)
- [ ] Sync context
- [ ] Dashboard context
- [x] Database migrations

## UI Checklist

- [x] Minimal monochrome design system (CSS variables)
- [x] shadcn/ui components initialized
- [x] AppShell (sidebar desktop + sheet mobile)
- [x] BottomNav (mobile)
- [x] SidebarNav (desktop)
- [x] WidgetCard component
- [x] Dashboard grid layout
- [x] Finance widgets
- [x] Habits widgets
- [x] Feedback components (offline banner, sync badge, failed-items panel; toasts not built)

## Testing Checklist

- [x] Vitest configured with jsdom
- [x] Testing Library configured
- [x] Playwright installed
- [x] MSW installed
- [x] First unit test
- [x] First component test (auth verify route)
- [x] First e2e test (CI: Playwright webServer, binary mode)
- [x] Go table-driven tests
- [x] Go httptest handler tests

## Completed Work

### Phase 0 — OpenCode Setup
- opencode.jsonc workspace config (Plan/Build agents, MCPs)
- 9 repo-local OpenCode skills
- BraveSearch, Context7, Filesystem MCPs

### Phase 1A — Root Scaffold
- git init, .gitignore
- pnpm workspace, root package.json, mise.toml
- docker-compose.yml (Redis)
- Go module with chi, health endpoint, embed handler
- Full DDD directory structure (9 bounded contexts)
- scripts/copy-web-assets.sh

### Phase 1B — Frontend Scaffold
- Vite React-TS app (@my/web)
- TanStack Router + Query + Zod + React Hook Form + Zustand
- shadcn/ui (new-york, 30+ components)
- Tailwind CSS v4 with @theme block
- 5 route files (root, index, login, authenticated, dashboard)
- ESLint, Vitest, Playwright configured
- Full pipeline validated: build → copy → Go binary (10MB)

### Phase 1C — Docs and CI
- README.md, ROADMAP.md
- Architecture docs (8 files)
- GitHub Actions CI (3 workflows)

### Phase 2 — Single-User Magic Link Auth
- Magic link request + verify flow (15-min expiry tokens)
- HttpOnly session cookie + CSRF double-submit pattern
- Auth middleware + CSRF middleware with tests
- TanStack Router pathless `_authenticated` layout with `beforeLoad` guard
- Login page + verify page (useRef guard for StrictMode)

### Phase 3 — Dashboard Widget System
- Widget registry (idempotent, side-effect imports)
- WidgetCard + WidgetErrorBoundary components
- DashboardGrid with responsive col-span sizing
- 5 widgets: Today Overview, Quick Expense, Recent Transactions, Habit Streak, Activity Heatmap

### Phase 4 — Finance Vertical Slice
- Transaction domain (expense/income, validation, amount_cents)
- Transaction service + libSQL repository
- Finance HTTP handler (CRUD + today-total)
- Frontend: schemas, API client, query keys, hooks, components

### Phase 5 — Habit Tracking Vertical Slice
- Habit domain (daily/weekly frequency, palette color tokens, streaks)
- Habit service + libSQL repository (toggle, archive, completions)
- Habits HTTP handler (CRUD + toggle + completions map)
- Frontend: schemas, API, hooks, contribution graph, habit cards
- 12-week GitHub-style activity heatmap (combined all habits)

### Phase — Tests + DX
- Air hot-reload with browser-sync proxy on :8090
- 11 Go test files (domain, service, middleware, datetime parsing)
- 8 frontend test files, 102 tests (schemas, api client, registry, query keys)
- mise seed task for deterministic dev data
- CI: 3 GitHub Actions workflows (API, Web, Fullstack)

### Phase — Finance Agent (Analytics + MCP + Skills)
- Derived analytics: anomalies, recurring charges, bill reconciliation,
  emergency fund, affordability, monthly digest (deterministic, per-currency)
- MCP surface: 41 tools (22 read + 19 write), semantic prompts, classify tool
- End-user skills in `skills/` (7 skills, standard Agent Skills conventions)
- Finance agent profile in `agent/finance/` (SOUL + Hermes wiring)
- Dashboard analytics overview consuming the same endpoints (no parallel math)

### Phase 6 (WIP) — Polish + Offline/Sync
- Mobile bottom navigation bar
- Route-level loading skeletons + error components
- IndexedDB mutation queue
- Network status store + sync engine
- Offline-aware API client wrapper
- Sync status UI component
- PWA workbox runtime caching for API routes

### Phase 8 — Production Docker Image
- Multi-stage Dockerfile: node (Vite) → Go (embed + build) → distroless/static:nonroot
- Binary self-contained: frontend assets, SQL migrations, and tzdata embedded
- docker-compose.yml: api + redis stack, named volumes, 20s stop grace
- GHCR publishing on v* tags (buildx, amd64+arm64, provenance) with a smoke test
- Runtime fixes it unlocked: embedded migrations (no silent empty schema),
  tzdata import (boots on minimal images), POSIX copy-web-assets.sh

### Phase 9 — E2E Tests in CI
- Playwright suite runs against the production binary (binary mode: no Vite, no proxy)
- Verify-route bug fixed en route: idle mutation state rendered a blank frame;
  component test added (first .tsx test in the repo)
- Environment parameterized (E2E_EMAIL, E2E_BASE_URL, E2E_MAILPIT_URL);
  fixed sleeps and Tailwind-class selectors removed (accessible labels instead)
- e2e-ci.yml: Redis + Mailpit services, fresh e2e.db, report artifact on failure
- MY_MAGIC_LINK_RATE raised for the test env (suite needs 7 links vs default 6)

### Phase 10 — Offline Correctness
- Dead-letter state for failed sync mutations: 4xx and retry-exhausted
  entries are kept (never silently deleted), with per-item Retry/Discard
  in the sync panel; unparseable entries parked in a separate corrupt
  store
- Post-drain reconciliation: TanStack Query invalidation + SW api-cache
  purge after a successful drain (kills the stale-list duplicate trap)
- Habit check-in made idempotent: optional `completed` set-state on the
  toggle endpoint (non-breaking; MCP flip unchanged), habit card sends
  explicit state + frozen date through the offline queue
- First sync tests (13): queue persistence/FIFO/dead-letter/corrupt
  parking + drain state machine; found and fixed a real corrupt-store
  runtime bug
- ROADMAP corrections: queue, sync indicator, feedback components

### Phase 11 — MVP Hardening (operational readiness)
- Backup + export: `VACUUM INTO` snapshots (`mise run backup`, `-backup`
  flag, authenticated `GET /api/v1/backup`), JSON export of all 12
  user-data tables (magic_tokens excluded); live restore drill passed
- SQLite pragmas: WAL, 5s busy timeout, foreign keys ON, single-writer
  pool — contention-safe API+MCP, cascades now live
- Explicit child deletes for goals/bills + `foreign_key_check` orphan
  diagnostics at boot
- Errors surfaced: Toaster mounted, login-form failure branch, mutation
  onError toasts, "saved offline" notice for queued changes
- PWA installability: 192/512/maskable icons + apple-touch-icon
- Deploy sanity: MY_WEB_URL required (no more localhost magic links),
  `/api/v1/ready` readiness probe, insecure-cookie boot warning,
  sliding session expiry, `-login-link` SMTP-down escape hatch
- Deployment docs: backup/restore runbook, probes, WAL caveats

### Release v1.1.0
- 64 commits since v1.0.2: Phases 7-11 (observability, production
  Docker, e2e in CI, offline correctness, MVP hardening) plus release
  tooling (release-guard preflight, HANDOFF.md removal)
- First-ever CI runs for api-ci/web-ci/fullstack-ci/e2e-ci/docker.yml
  and the release + GHCR pipelines; goreleaser snapshot verified via
  `mise run release:check`

### Release v1.1.1
- Fix finance pages crashing on private HTTP origins where
  `crypto.randomUUID` is unavailable (non-secure context). New shared
  `randomUUID()` utility falls back to RFC 4122 v4 via
  `crypto.getRandomValues`; idempotency keys unchanged.

### Release v1.2.0
- GCash PDF statement import: privacy-first client-side parsing (pdf.js),
  atomic backend batch commit, fingerprint replay protection, dependency-safe
  rollback, import history/undo, wizard with wallet selection or creation and
  reviewed transfer mapping (migration 011)
- Confidence-gated LLM analysis (`intelligence` context): OpenAI
  Responses/Codex, OpenAI-compatible (Yunwu) and Ollama providers, optional
  sandboxed Codex CLI adapter, Brave/Exa MCP web-search connectors,
  encrypted credentials (AES-256-GCM, env master key), per-field calibrated
  confidence with preselect/review/unresolved buckets; suggestions never
  commit finance data (migration 012)
- Settings page: provider + web-search configuration with credential
  replacement and connection tests; status endpoint; import page shows
  availability card
- Security hardening: SSRF DNS + redirect defense, feature-flag enforcement,
  privacy-safe agent-run summaries (counts only), calibrated search gating

### Release v1.2.1
- Fix GCash PDF import crashing on private HTTP/tailnet origins where
  `crypto.subtle` is undefined (non-secure context). File fingerprinting now
  prefers Web Crypto and falls back to lazily-loaded audited @noble/hashes;
  known-vector + insecure-origin regression tests added.

## Remaining Work

- Phase 6: Offline/sync backend — server sync context (push/pull/status),
  revision columns/tombstones, batch drain, conflict policy (client queue,
  dead-letter, invalidation, idempotent habit check-in now exist)
- Known gaps: `rediss://` TLS Redis, down-migrations, SQLite session
  fallback, transaction date bounds, toasts beyond the main dialogs

## Known Issues

- shadcn calendar component has `@ts-expect-error` for `table` property (react-day-picker type mismatch)
- Tailwind v4 dynamic arbitrary classes don't work at runtime — use inline styles with CSS var refs
- `GetAllCompletionsGrouped` TotalHabits counts only habits with completions in range (not total)