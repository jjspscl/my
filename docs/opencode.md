# OpenCode Skills

## Setup

All skills are repo-local in `.opencode/skills/`.

## Available Skills

| Skill | When to Use |
|---|---|
| caveman-ultra | Default -- terse communication |
| repo-architect | Architecture decisions, file placement |
| frontend-shadcn | UI components, styling |
| frontend-zod-contracts | Schema definitions, type inference |
| tanstack-router-query | Routing, server state, loaders |
| go-ddd-api | Backend context implementation |
| offline-pwa-sync | PWA, offline, sync features |
| test-quality | Writing tests |
| security-review | Security auditing |

## Agents

- **plan** (Claude Opus 4.6): Architecture, review, approval. Read-only.
- **build** (DeepSeek V4 Flash): Implementation, no step limit. Full write access.

## Workflow

1. Plan agent writes instructions to `HANDOFF.md`
2. Switch to Build agent
3. Build agent reads and executes `HANDOFF.md`
4. Build marks `HANDOFF.md` as completed
5. Switch back to Plan for review

## MCPs

- **brave-search**: Web search
- **context7**: Library documentation lookup
- **filesystem**: Project file access