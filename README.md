# my

A modular personal dashboard for finance, habits, and daily life.

> Your day, money, and habits in one calm dashboard.

## Stack

- **Backend**: Go, chi, libSQL (embedded/Turso), Redis
- **Frontend**: React, Vite, TypeScript, TanStack Router, TanStack Query, Zod, shadcn/ui, Tailwind CSS
- **Architecture**: DDD modular monolith, single-binary production deployment
- **Offline**: PWA, IndexedDB mutation queue, service worker

## Quick Start

```bash
# Install tools (requires mise)
mise install

# Install dependencies
mise run install

# Start development servers (includes Redis + Mailpit)
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
| `mise run build` | Build production binary with embedded frontend |
| `mise run test` | Run all tests |
| `mise run lint` | Run all linters |
| `mise run typecheck` | Run frontend type checking |
| `mise run clean` | Clean build outputs |

## Architecture

```
my/
  apps/api/          Go API server (chi, libSQL, DDD contexts)
  apps/web/          React frontend (Vite, TanStack, shadcn/ui)
  packages/          Shared packages
  infrastructure/    Docker, K8s, Terraform
  docs/              Documentation
  .opencode/         AI coding agent skills
```

## Production

In production, `my` runs as a single Go binary:

1. Frontend is compiled to static assets by Vite
2. Assets are embedded into the Go binary via `go:embed`
3. Go serves both API routes and frontend from one HTTP server
4. No Node.js or Vite in production

## Development

In development, two servers run independently:

- **Go API** on `localhost:8080` — handles API routes
- **Vite** on `localhost:5173` — handles React HMR, proxies `/api` to Go

### Dev services

`mise run dev` automatically starts required dependencies using Docker:

- **Redis** (`my-redis`) — session storage, port 6379
- **Mailpit** (`my-mailpit`) — local SMTP server for email testing, port 1025

Mailpit provides a web UI at **http://localhost:8025** where you can view emails sent by the app (magic link, etc.).

If a container is already running, it is reused. If it exists but is stopped, it is restarted. If it does not exist, it is created.

## Modules

- **Finance** — accounts, transactions, budgets, categories, recurring expenses, savings goals
- **Habits** — daily/weekly habits, check-ins, streaks, reminders

## License

MIT
