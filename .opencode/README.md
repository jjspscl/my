# OpenCode assets — `my`

Repo-local OpenCode support files for the `my` personal dashboard project.

## What lives here

- `skills/*/SKILL.md` — repo-local reusable skills
- local npm package metadata for OpenCode-related tooling

Project-wide OpenCode config lives in root `opencode.jsonc`.
Project-wide agent rules live in root `AGENTS.md`.

## Agents

| Agent | Role |
|---|---|
| `plan` | default review/scoping agent |
| `build` | implementation agent |

## Custom command

`opencode.jsonc` defines a `build` command that executes the plan approved in the conversation (no repo handoff file).

Session handoffs are temporary execution state: Engram summaries + temp files, never durable architecture documentation.

## Skills

| Skill | Purpose |
|---|---|
| caveman-ultra | terse engineering communication |
| repo-architect | monorepo architecture and file placement rules |
| frontend-shadcn | UI styling/component guidance |
| frontend-zod-contracts | schema-first frontend contracts |
| tanstack-router-query | router/query conventions |
| go-ddd-api | Go DDD/backend slice guidance |
| offline-pwa-sync | offline/PWA/sync guidance |
| test-quality | focused testing guidance |
| security-review | security review checklist |

## MCPs in project config

### Enabled

- `brave-search`
- `context7`
- `filesystem`
- `playwright`
- `gh_grep`

Use higher-cost MCPs only when the task actually needs browser automation or public code example lookup.

## Recommended reading order

1. `AGENTS.md`
2. `docs/opencode.md`
3. subsystem docs in `docs/`
