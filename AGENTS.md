# my — agent rules

Personal dashboard monorepo: Go API + React/Vite frontend; production ships one Go binary with embedded Vite assets.

## Context protocol

Engram is primary durable agent memory; git docs remain portable backup.

1. At session start or after compaction/reset, call `mem_context`.
2. Before planning or changing a subsystem, use `mem_search` for relevant decisions, bugs, and conventions.
3. Save significant decisions, fixes, discoveries, conventions, config, and architecture changes immediately with `mem_save`. Prefer stable `topic_key` values for evolving truth.
4. Before reporting significant work complete, call `mem_session_summary`.
5. Treat memory as recall, not proof. Verify stale or consequential claims against code and git docs.

Never store secrets, tokens, personal finance records, or other sensitive values in memory.

## Source order

1. This file: workflow and safety.
2. Engram: recent work and durable project decisions.
3. Relevant code and tests: implementation truth.
4. Lazy-read docs: `docs/architecture.md`, `docs/backend-ddd.md`, `docs/frontend.md`, `docs/offline-sync.md`, `docs/opencode.md`.
5. `ROADMAP.md`: future work only.
6. `HANDOFF.md`: temporary plan→build task state; never durable architecture.

Do not preload deeper docs. Read only files relevant to current task.

## Canonical commands

Run from repo root: `mise run dev|build|build:mcp|test|lint|typecheck|migrate|seed|clean|release:check|release:tag`.

Production: Vite assets copy to `apps/api/internal/platform/web/static/`, then embed in `bin/my`. Development: API `:8080`, Vite `:5173`, Redis `:6379`, Mailpit SMTP `:1025` / UI `:8025`.

MCP: main binary serves optional bearer-protected `/mcp`; standalone `bin/my-mcp` serves stdio. `MY_MCP_ENABLED=false` by default. Full setup lives in `docs/mcp.md`.

## Workflow

- `plan` scopes/reviews; `build` implements exactly specified work.
- Before commit, push, or PR work, load `git-commit-and-push`.
- Load subsystem skills only when task needs them.
- Update durable docs when runtime, architecture, setup, or workflow truth changes.
- Preserve unrelated working-tree changes.

## Safety

- Never commit `.env`, PATs, auth tokens, secrets, or personal data.
- Keep finance/data tooling conservative and changes minimal/reversible.
- Current auth is magic-link plus Redis opaque session cookie and double-submit CSRF, not JWT.
- Offline sync is partial; never present it as complete end-to-end behavior.
