# Backend DDD

## Module Path

`github.com/jjspscl/my`

## Stack

- Go 1.26+
- chi (HTTP router)
- libSQL (embedded SQLite / Turso remote)
- sqlc (type-safe SQL)
- goose (migrations)
- slog (structured logging)

## Bounded Contexts

Each context is self-contained with:
- Domain: entities, value objects, aggregate roots, domain events, repository ports
- Application: commands, queries, application services
- Infrastructure: repository implementations, external adapters
- Interfaces/HTTP: handlers, DTOs, route registration

## Rules

1. Domain layer imports NO external packages (no HTTP, DB, framework)
2. Application layer depends only on domain ports
3. Infrastructure implements domain ports
4. Handlers are thin: validate input -> call application -> map output
5. Domain entities are never API response bodies
6. Each context registers its own routes

## Database

- Default: embedded libSQL (file-based SQLite)
- Optional: Turso remote (set MY_TURSO_URL + MY_TURSO_AUTH_TOKEN)
- Migrations via goose (SQLite dialect)
- Queries via sqlc (SQLite dialect)

## Auth

- JWT access tokens in HttpOnly cookies
- CSRF protection (double-submit or synchronizer token)
- Refresh token rotation
- Works in dev mode (localhost cookies)