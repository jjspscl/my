# OpenCode in `my`

## Source of truth

OpenCode reads project rules from the root `AGENTS.md`.

This file complements `AGENTS.md` with repo-specific OpenCode config details. It is loaded through `opencode.jsonc` `instructions` so keep it focused on workflow/config, not general project prose.

## Config layout

Project OpenCode config lives in:

- `opencode.jsonc` — project config
- `.opencode/skills/*/SKILL.md` — repo-local skills
- `AGENTS.md` — project agent rules
- `HANDOFF.md` — temporary task handoff state

## Rules precedence

Relevant OpenCode behavior:

- root `AGENTS.md` has priority over project `CLAUDE.md`
- global `~/.config/opencode/AGENTS.md` can still apply as user-level rules
- `instructions` from `opencode.jsonc` are combined with `AGENTS.md`

## Agents

### `plan`

Current default agent.

Role:

- scope work
- prepare/review `HANDOFF.md`
- run validation and review output
- avoid direct product code edits unless task coordination requires updating docs or `HANDOFF.md`

### `build`

Execution agent.

Role:

- implement requested changes
- follow architecture and handoff instructions
- report changed files, validation results, and issues

## Workflow

Typical project workflow:

1. `plan` analyzes request
2. `plan` decides whether `HANDOFF.md` needs task-specific instructions
3. `build` executes implementation work
4. `plan` reviews files and validation output
5. durable docs get updated when runtime/architecture truth changes

`HANDOFF.md` is not durable project documentation. Do not treat it as long-term truth.

## Skills

Repo-local skills currently available:

- `caveman-ultra`
- `repo-architect`
- `frontend-shadcn`
- `frontend-zod-contracts`
- `tanstack-router-query`
- `go-ddd-api`
- `offline-pwa-sync`
- `test-quality`
- `security-review`

Load them on demand. Do not assume every task needs every skill.

## MCP inventory

### Enabled in project config

- `brave-search` — current-state web search
- `context7` — library/framework docs lookup
- `filesystem` — repo-scoped file access
- `playwright` — browser automation for PWA/offline/browser flows
- `gh_grep` — GitHub code search examples

Reason: MCP servers add context cost. Browser automation and broad code search should still be used only when the task needs them.

## MCP guidance

- use `context7` before generic web search when asking library/API questions
- use `brave-search` for recent docs, blog posts, or ecosystem/tooling research
- use `playwright` only for browser-state or PWA verification tasks
- use `gh_grep` when example code from public repos is more useful than prose docs

If Playwright MCP needs an extension token, provide it locally through `PLAYWRIGHT_MCP_EXTENSION_TOKEN`. Never commit the real value.

Useful OpenCode commands when using remote MCP auth:

- `opencode mcp list`
- `opencode mcp auth <server>`
- `opencode mcp logout <server>`
- `opencode mcp debug <server>`

## Config notes

Current project config choices:

- `default_agent` is `plan`
- `share` is disabled because repo context can include personal finance/life data
- watcher ignores noisy build directories
- compaction is enabled with pruning to keep long sessions usable

## Workspace notes

This repo does not keep committed editor-specific workspace config.

If you use an IDE MCP host locally, keep that config in your own user-level settings and never commit secrets.
