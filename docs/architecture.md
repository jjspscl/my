# Architecture

## Overview

`my` is a modular monolith personal dashboard with a Go backend and React frontend.

## Production Runtime

```
┌─────────────────────────────┐
│        Go Binary (my)       │
├─────────────────────────────┤
│  chi Router                 │
│  ├── /api/v1/* → handlers   │
│  └── /* → embedded SPA      │
├─────────────────────────────┤
│  Embedded Static Assets     │
│  (compiled React/Vite)      │
├─────────────────────────────┤
│  libSQL (embedded SQLite)   │
│  Redis (cache/sessions)     │
└─────────────────────────────┘
```

## Development Runtime

```
┌──────────────┐     ┌──────────────┐
│  Vite :5173  │────▶│  Go API :8080│
│  React HMR   │proxy│  chi router  │
│  /api → 8080 │     │  libSQL      │
└──────────────┘     └──────────────┘
```

## Backend Bounded Contexts

| Context | Responsibility |
|---|---|
| identity | User registration, profiles |
| access | Authentication, authorization, sessions |
| dashboard | Layouts, widgets, preferences |
| modules | Module registry, capabilities |
| finance | Accounts, transactions, budgets |
| habits | Habits, check-ins, streaks |
| sync | Offline sync, conflict resolution |
| notifications | Reminders, alerts |
| health | System health checks |

## Context Structure

Each bounded context follows clean architecture:

```
contexts/<name>/
  domain/          Entities, value objects, events, ports
  application/     Commands, queries, services
  infrastructure/  Repository implementations, adapters
  interfaces/http/ Handlers, DTOs, routes
```

## Frontend Feature Structure

```
features/<name>/
  schemas/     Zod schemas (source of truth for types)
  api/         Query keys, queries, mutations
  components/  Feature-specific UI
  hooks/       Feature-specific hooks
  lib/         Business logic utilities
```

## Key Principles

- Domain layer has zero external dependencies
- Handlers are thin (validate -> delegate -> respond)
- DTOs never expose domain entities directly
- All frontend types inferred from Zod schemas
- Server state in TanStack Query, never Zustand
- URL state in TanStack Router, never Zustand