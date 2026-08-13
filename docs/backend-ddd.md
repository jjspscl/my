# Backend DDD

## Module path

`github.com/jjspscl/my`

## Current backend stack

- Go 1.26+
- chi
- libSQL client + embedded SQLite via `modernc.org/sqlite`
- Redis via `go-redis/v9`
- `slog` JSON logging
- SMTP sender for magic-link delivery

## Current reality

The backend is a DDD-inspired modular monolith.

Implemented application slices today:

- `access`
- `finance`
- `habits`

MCP is an inbound adapter in `internal/platform/mcp`, not a fourth bounded
context. Its handlers call application services directly and inject the
configured single-user email server-side.

Other context folders are scaffolds/placeholders, not full production slices yet.

## Layering rules

1. Domain layer should stay free of HTTP/DB/framework concerns.
2. Application layer coordinates use cases over domain ports.
3. Infrastructure implements repository/adaptor details.
4. HTTP handlers stay thin: decode/validate -> call application -> map response.
5. API responses should not expose domain entities directly without mapping.

## Current auth model

Current auth is **single-user magic-link auth**, not JWT/refresh-token auth.

Flow:

1. client requests a magic link
2. backend issues a short-lived token and sends email
3. client verifies token
4. backend creates an opaque session in Redis
5. backend sets `my_session` cookie and JS-readable `my_csrf` cookie

Protected routes use:

- Redis-backed session lookup
- CSRF protection via `X-CSRF-Token`
- Session and CSRF cookies use configurable `Secure` behavior via `MY_SECURE_COOKIES`; when unset, it follows `MY_WEB_URL` scheme.

## Database

Current state:

- default local database: `file:my_dev.db`
- optional remote Turso/libSQL supported through env vars
- migrations stored as SQL files under `apps/api/migrations/`
- migrations executed by project code during startup / migrate command

This repo currently does **not** use generated `sqlc` repositories in the active implementation.

## Router shape

Current API routes live under `/api/v1`.

Implemented areas:

- `/api/v1/health`
- `/api/v1/auth/*`
- `/api/v1/finance/*`
- `/api/v1/habits/*`
- optional `/mcp` streamable HTTP endpoint on dedicated `MY_MCP_BIND:MY_MCP_PORT`, protected by bearer token

## Logging and middleware

Current middleware/runtime behavior includes:

- request IDs — generated and stored in the context; request and error
  log lines both carry them (never trusted from the client header)
- real IP middleware
- panic recovery — logs the stack trace, responds 500
- structured request logging — JSON via slog (method, path, status,
  duration, request_id)
- structured error logging — `response.WriteError` emits JSON records
  via slog (status, method, path, request_id, client_msg, cause);
  Error level when a cause exists, Warn for client-caused 4xx
- magic-link rate limiting — in-memory sliding window per IP
  (`MY_MAGIC_LINK_RATE` requests per 15 min), 429 + Retry-After;
  applied only to `POST /auth/magic-link`

Redis is used for session storage.
