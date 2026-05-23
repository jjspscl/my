---
name: go-ddd-api
description: Build the Golang API using chi, libSQL, sqlc, DDD-inspired clean architecture, and modular monolith bounded contexts.
compatibility: opencode
---

# go-ddd-api

Use:
- Go
- chi
- libSQL (embedded default, Turso remote optional)
- sqlc
- goose migrations
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
