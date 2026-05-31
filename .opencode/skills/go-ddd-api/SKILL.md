---
name: go-ddd-api
description: Build the Golang API using chi, libSQL, DDD-inspired clean architecture, and modular monolith bounded contexts.
compatibility: opencode
---

# go-ddd-api

Use:
- Go
- chi
- libSQL (embedded default, Turso remote optional)
- manual SQL repositories in infrastructure layer
- SQL migration files under `apps/api/migrations/`
- clean architecture
- hexagonal boundaries
- modular monolith DDD

Each context uses:
- domain
- application
- infrastructure
- interfaces/http

Domain layer has no:
- HTTP imports
- database imports
- framework imports
- logging side effects

Handlers stay thin.
Domain entities never become API responses directly.

Current repo auth model:
- magic-link auth
- Redis-backed opaque session cookie
- CSRF cookie/header protection

Do not assume JWTs, refresh tokens, sqlc, or goose unless task explicitly adds them.
