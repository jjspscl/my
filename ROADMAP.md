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
- [ ] Offline mutation queue
- [ ] Sync status indicator

## Architecture Checklist

- [x] Go API with chi router
- [x] SPA fallback handler with go:embed
- [x] DDD bounded context directory structure (9 contexts)
- [x] Frontend feature-based directory structure
- [x] libSQL database integration (embedded mode via modernc.org/sqlite)
- [x] SQL migrations
- [x] Repository pattern (manual SQL)
- [x] Session cookies + CSRF double-submit
- [ ] Request middleware (ID, panic recovery, CORS, rate limit)
- [ ] Structured JSON logging
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
- [ ] Zustand ephemeral UI stores
- [ ] Offline IndexedDB mutation queue
- [ ] React Hook Form + zodResolver forms

## Backend Checklist

- [x] Go module (github.com/jjspscl/my)
- [x] chi router with middleware
- [x] Health endpoint (/api/v1/health)
- [x] Static frontend embed + SPA handler
- [x] libSQL connection (embedded mode)
- [x] Config from environment
- [ ] Structured logger (slog)
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
- [ ] Feedback components (offline banner, sync badge, toasts)

## Testing Checklist

- [x] Vitest configured with jsdom
- [x] Testing Library configured
- [x] Playwright installed
- [x] MSW installed
- [x] First unit test
- [ ] First component test
- [ ] First e2e test
- [x] Go table-driven tests
- [ ] Go httptest handler tests

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

## Remaining Work

- Phase 6: Offline/sync infrastructure (in progress)
- Phase 7: Additional middleware (structured logging, panic recovery, rate limit)
- Phase 8: Docker production image
- Phase 9: E2E tests (Playwright)

## Known Issues

- shadcn calendar component has `@ts-expect-error` for `table` property (react-day-picker type mismatch)
- Tailwind v4 dynamic arbitrary classes don't work at runtime — use inline styles with CSS var refs
- `GetAllCompletionsGrouped` TotalHabits counts only habits with completions in range (not total)