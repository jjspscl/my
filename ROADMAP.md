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
- [ ] Auth/session shell (registration, login, logout, refresh)
- [ ] Dashboard widget registry
- [ ] Finance quick expense
- [ ] Monthly spend widget
- [ ] Habit check-in
- [ ] Today habits widget
- [ ] Offline mutation queue
- [ ] Sync status indicator

## Architecture Checklist

- [x] Go API with chi router
- [x] SPA fallback handler with go:embed
- [x] DDD bounded context directory structure (9 contexts)
- [x] Frontend feature-based directory structure
- [ ] libSQL database integration (embedded + Turso)
- [ ] goose migrations
- [ ] sqlc code generation
- [ ] JWT + HttpOnly cookies + CSRF
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
- [ ] Zod schema contracts (finance, habits, dashboard, auth)
- [ ] Query key factories per feature
- [ ] Route loaders with ensureQueryData
- [ ] Zustand ephemeral UI stores
- [ ] Offline IndexedDB mutation queue
- [ ] React Hook Form + zodResolver forms

## Backend Checklist

- [x] Go module (github.com/jjspscl/my)
- [x] chi router with middleware
- [x] Health endpoint (/api/v1/health)
- [x] Static frontend embed + SPA handler
- [ ] libSQL connection (embedded mode)
- [ ] Config from environment
- [ ] Structured logger (slog)
- [ ] Auth context (identity, access)
- [ ] Finance context (domain, application, infra, HTTP)
- [ ] Habits context (domain, application, infra, HTTP)
- [ ] Sync context
- [ ] Dashboard context
- [ ] Database migrations

## UI Checklist

- [x] Minimal monochrome design system (CSS variables)
- [x] shadcn/ui components initialized
- [ ] AppShell (mobile/tablet/desktop)
- [ ] BottomNav (mobile)
- [ ] SidebarNav (desktop)
- [ ] WidgetCard component
- [ ] Dashboard grid layout
- [ ] Finance widgets
- [ ] Habits widgets
- [ ] Feedback components (offline banner, sync badge, toasts)

## Testing Checklist

- [x] Vitest configured with jsdom
- [x] Testing Library configured
- [x] Playwright installed
- [x] MSW installed
- [ ] First unit test
- [ ] First component test
- [ ] First e2e test
- [ ] Go table-driven tests
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

## Remaining Work

- Phase 2: Auth shell (backend + frontend)
- Phase 3: Dashboard widget system
- Phase 4: Finance vertical slice (quick expense)
- Phase 5: Habits vertical slice (check-in)
- Phase 6: Offline/sync infrastructure
- Phase 7: CI/CD pipelines
- Phase 8: Documentation
- Phase 9: Docker production image

## Known Issues

- `pnpm@latest` causes cosmetic warnings (fixed with pinned version)
- shadcn calendar component has `@ts-expect-error` for `table` property (react-day-picker type mismatch)
- `vitest` exits with code 1 when no test files exist (expected, will resolve when first test is added)
- Tailwind v4 doesn't support `@apply` with custom CSS variable utilities (resolved by using plain CSS + `@theme` block)