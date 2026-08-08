# HANDOFF — Track 2: MCP Server + Release Pipeline

Status: **implemented locally, not committed**. Validation passed; release tag not created.

Goal: expose `my`'s dashboard/API resources to external coding agents (Codex, Claude Code, opencode, Hermes, OpenClaw) through a Model Context Protocol server, and ship it as a versioned, agent-installable release artifact.

Scope: backend-only Go work, plus CI/release config, scripts, and docs. **No frontend changes. No DB migration. No change to existing `/api/v1/*` behavior.**

Branch: `feat/mcp-server` off `main`. Working tree contains this track's implementation and prior handoff edits.

Implementation note: actual SDK v1.7.0 uses `ToolAnnotations.DestructiveHint *bool`; GoReleaser v2.17.1 uses archive `ids` and `formats`. Standalone stdio skips migrations and verifies `_migrations` exists, returning an actionable error otherwise.

---

## Locked decisions — do not re-litigate

| Axis | Decision |
|---|---|
| MCP auth | static bearer token `MY_MCP_TOKEN`, `/mcp` bound to `127.0.0.1` |
| Tool surface | full CRUD across access/finance/habits |
| Transports | streamable HTTP **and** stdio |
| MCP features | tools + resources + prompts |
| Go SDK | `github.com/modelcontextprotocol/go-sdk` **v1.7.0** |
| Release tool | GoReleaser |
| Platforms | linux + darwin, amd64 + arm64. No Windows. |
| Artifacts | `my` and `my-mcp` as separate archives |
| Update path | `--check-update` prints newer tag. **No self-replacing binary.** |
| First tag | `v1.0.0` |
| Install UX | `curl \| sh` script, checksum + attestation verified, prints client config but **never edits client configs** |

Verified facts the plan depends on:

- Go SDK v1.7.0 exists and is the latest release; repo is already on Go 1.26.2, which satisfies it.
- All five target clients support remote streamable-HTTP MCP with a bearer header.
- Every application service method takes `userEmail` explicitly, so MCP can call services in-process. **Do not make HTTP self-calls from the MCP layer.**
- Repo has zero git tags, no release workflow, no goreleaser config, and no version embedded in any binary.
- No MCP code exists in `apps/api`. `grep -i mcp` over Go files returns nothing. This is greenfield.

Known tradeoff, accepted by the user: starting at `v1.0.0` means renaming or removing a tool later is technically a breaking change. Do not add a compatibility shim layer to work around this. Ship the surface as designed.

---

## Phase ordering

Phases are sequential. Each maps to one commit. Do not reorder — Phase 1 is a prerequisite for Phase 3, and Phase 2 must land before Phase 6 so version strings exist to embed.

Run `mise run lint` and `mise run test` after every phase, not just at the end.

---

## Commit 1 — `refactor(api): extract bootstrap wiring from main`

Behavior-preserving refactor. No new features. This must be a clean no-op at runtime.

### Problem

`apps/api/cmd/api/main.go` wires everything inline in `main()`: config, db open, migrate, redis, session store, mailer, 8 repos, 8 services, 8 handlers. The MCP stdio entrypoint (`cmd/mcp`) needs the exact same dependency graph minus the HTTP handlers. Duplicating that wiring guarantees drift.

### 1A. New bootstrap package

New file: `apps/api/internal/platform/bootstrap/bootstrap.go`

```go
package bootstrap

type App struct {
    Cfg      *config.Config
    Log      *slog.Logger
    DB       *sql.DB          // match the concrete type database.Open returns
    Redis    *redis.Client    // match predis.NewClient's return type
    Sessions session.Store

    Auth     *accessapp.AuthService
    Tx       *financeapp.TransactionService
    Budget   *financeapp.BudgetService
    Bill     *financeapp.BillService
    Goal     *financeapp.GoalService
    Wallet   *financeapp.WalletService
    Transfer *financeapp.TransferService
    Habit    *habitapp.HabitService
}

func New(cfg *config.Config, log *slog.Logger) (*App, error)
func (a *App) Close() error
```

`New` performs, in this order (preserving current `main.go` semantics exactly):

1. `database.Open(cfg.DatabaseURL)`
2. `database.Migrate(db, "migrations")`
3. `predis.NewClient(cfg.RedisURL)`
4. `session.NewRedisStore(rdb, cfg.SessionTTL)`
5. `mail.NewSMTPSender(...)`
6. repos, then services

Critical detail: `txSvc.WithBillAutoMatcher(billSvc)` at `main.go:75` must be preserved. It is easy to drop during extraction and there is no test covering it. Add one (see 1C).

`Close()` closes redis then db, joining errors rather than returning only the first. On partial failure in `New`, close whatever was already opened before returning the error — the current code relies on `defer` in `main`, which will no longer apply.

Migration path note: `database.Migrate(db, "migrations")` uses a **relative** path. `cmd/mcp` will run from a different working directory when installed as a standalone binary. Confirm how migrations resolve and report what you find. Do **not** silently change the path in this commit — flag it, and handle it in Phase 4 where the stdio entrypoint is built.

### 1B. Slim main.go

`cmd/api/main.go` becomes: logger → `config.Load()` → `bootstrap.New` → build handlers from `app.*` services → `newRouter(...)` → `http.ListenAndServe`.

Handler construction (`accesshttp.NewAuthHandler`, the 7 finance/habit handlers) stays in `cmd/api`. Handlers are HTTP-transport concerns and the MCP layer must not depend on them.

`routerDeps` and `newRouter` in `cmd/api/router.go` stay unchanged.

### 1C. Tests

New `apps/api/internal/platform/bootstrap/bootstrap_test.go`:

- `New` returns a non-nil service on every exported field (guards against a field added to the struct but never wired)
- bill auto-matcher is attached to the transaction service
- `Close()` is safe to call twice

Bootstrap needs real db + redis. Check how existing tests handle this — `MY_TEST_DATABASE_URL` exists in `.env.example`. If redis is unavailable in CI, skip with `t.Skip` on connection failure rather than failing the suite, and say so in your report.

`cmd/api/router_test.go` must still pass untouched. If it needs changes, the refactor was not behavior-preserving — stop and report.

### Validate

```
mise run lint
mise run test
mise run build
```

Then `mise run dev` and confirm the API boots and a magic-link login still works. State explicitly whether you ran it.

---

## Commit 2 — `feat(api): embed version metadata in binaries`

Currently nothing in the repo reports a version.

### 2A. Version package

New file: `apps/api/internal/platform/version/version.go`

```go
package version

var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
)

func String() string  // e.g. "v1.0.0 (abc1234, 2026-08-08)"
```

Package-level vars set via `-X github.com/jjspscl/my/internal/platform/version.Version=...`. Using a shared package rather than `main.version` means both binaries get it from one ldflags block.

### 2B. Surface it

- `cmd/api`: `--version` flag prints `version.String()` and exits 0
- `/api/v1/health` response gains a `version` field: `{"status":"ok","version":"..."}`
- MCP `Implementation.Version` will use it in Phase 3

Health is currently a hardcoded string literal at `router.go:39-42`. Switch to `encoding/json` rather than hand-building the JSON.

### 2C. Tests

- `version.String()` formats correctly with defaults and with injected values
- health endpoint returns 200 with a `version` key (extend `router_test.go`)

### Validate

```
mise run lint
mise run test
go build -ldflags "-X github.com/jjspscl/my/internal/platform/version.Version=v9.9.9" -o /tmp/vtest ./cmd/api && /tmp/vtest --version
```

Confirm it prints `v9.9.9`. If ldflags injection silently fails, the whole release pipeline reports `dev` forever. Verify this before moving on, then `rm /tmp/vtest`.

---

## Commit 3 — `feat(api): add MCP server adapter with tools, resources, prompts`

The core of this track. No transport wiring yet — this commit only builds the `*mcp.Server` and its registrations, tested in-process.

### 3A. Dependency

```
cd apps/api && go get github.com/modelcontextprotocol/go-sdk@v1.7.0
```

Pin the exact version. Do not use `@latest`.

### 3B. Placement

New package: `apps/api/internal/platform/mcp/`

Rationale: MCP is an inbound transport adapter across all bounded contexts, not a bounded context of its own. Putting it in `internal/contexts/mcp/` would break the DDD meaning of that directory — every other entry there is a vertical slice with domain/application/infrastructure. Load the `repo-architect` skill and confirm this placement before writing files. If the skill disagrees, follow the skill and report the deviation.

Package name collides with the SDK's `mcp` package. Import the SDK as `mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"`.

Files:

- `server.go` — `NewServer(app *bootstrap.App, opts Options) *mcpsdk.Server`
- `tools_finance.go`
- `tools_habits.go`
- `resources.go`
- `prompts.go`
- `audit.go`

### 3C. Options

```go
type Options struct {
    ReadOnly bool   // from MY_MCP_READONLY
}
```

When `ReadOnly` is true, write tools must **not be registered at all** — not registered-and-rejecting. An agent should never see a tool it cannot call.

### 3D. Tools

All handlers use the SDK's typed tool registration so JSON Schema is generated from Go input structs. Do not hand-write schemas.

Every handler injects `app.Cfg.UserEmail` server-side. **No tool input struct may contain a user/email field.** A model must not be able to address another user's data, even though this is single-user today.

Read tools (10):

| Tool | Service call |
|---|---|
| `finance_list_transactions` | `Tx.List` |
| `finance_today_total` | `Tx.GetTodayTotal` |
| `finance_budget_summary` | `Budget.GetSummary` |
| `finance_list_bills` | `Bill.List` |
| `finance_upcoming_bills` | `Bill.GetUpcoming` |
| `finance_list_goals` | `Goal.ListSummaries` |
| `finance_list_wallets` | `Wallet.ListWithBalances` |
| `finance_list_transfers` | `Transfer.List` |
| `habits_list` | `Habit.ListWithStatus` |
| `habits_completions` | `Habit.GetAllCompletionsGrouped` |

Write tools (18):

| Tool | Service call | Destructive |
|---|---|---|
| `finance_create_transaction` | `Tx.Create` | no |
| `finance_delete_transaction` | `Tx.Delete` | **yes** |
| `finance_upsert_budget` | `Budget.UpsertBudget` | no |
| `finance_create_bill` | `Bill.Create` | no |
| `finance_update_bill` | `Bill.Update` | no |
| `finance_delete_bill` | `Bill.Delete` | **yes** |
| `finance_pay_bill` | `Bill.MarkPaid` | **yes** |
| `finance_create_goal` | `Goal.Create` | no |
| `finance_update_goal` | `Goal.Update` | no |
| `finance_delete_goal` | `Goal.Delete` | **yes** |
| `finance_add_goal_contribution` | `Goal.AddContribution` | no |
| `finance_create_wallet` | `Wallet.Create` | no |
| `finance_update_wallet` | `Wallet.Update` | no |
| `finance_archive_wallet` | `Wallet.Archive` | **yes** |
| `finance_create_transfer` | `Transfer.Create` | no |
| `habits_create` | `Habit.Create` | no |
| `habits_toggle` | `Habit.ToggleCompletion` | no |
| `habits_archive` | `Habit.Archive` | **yes** |

Destructive tools get `Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: true, IdempotentHint: false}`. Read tools get `ReadOnlyHint: true`.

`finance_pay_bill` is marked destructive because it writes a payment record that has no delete tool.

Tool descriptions matter — they are the only thing a model sees. Each needs one sentence on what it does plus any non-obvious argument semantics (date formats, currency minor units, month string format). Check the domain types for the actual formats; do not guess.

Error handling: service errors return as MCP tool errors with the service's message text, not as protocol-level failures. Do not wrap them in extra prose. Do not leak stack traces.

### 3E. Resources

Read-only, cheap ambient context:

- `my://wallets` — wallets with balances
- `my://budget/current` — current month budget summary
- `my://bills/upcoming` — next 14 days
- `my://habits/today` — today's habits with completion status
- `my://dashboard/snapshot` — composite: today total, upcoming bills count, habit completion ratio

`my://dashboard/snapshot` has no single backing service. Compose it in the MCP layer from existing service calls. Do **not** create a new `dashboard` bounded context for this — that is out of scope.

All resources JSON, `application/json` MIME type.

### 3F. Prompts

- `weekly_finance_review` — no args
- `budget_health_check` — optional `month` arg
- `habit_streak_report` — optional `days` arg, default 30

Prompt text should tell the model which tools to call, not embed data.

### 3G. Audit logging

`audit.go` wraps every tool handler. One `slog` line per call at info level:

- tool name
- duration
- outcome (ok / error)
- error message when failed

**Never log argument values or result payloads.** This is finance and personal-habit data. A leaked debug log is a data leak. This constraint is not negotiable — do not add a "verbose" mode that bypasses it.

### 3H. Tests

`apps/api/internal/platform/mcp/server_test.go`:

- registry completeness: `ListTools` returns exactly the 28 expected names
- `ReadOnly: true` ⇒ `ListTools` returns exactly the 10 read names, and zero write names
- every tool has a non-empty description and a generated input schema
- destructive tools carry `DestructiveHint: true`
- no tool's input schema contains a property named `user`, `email`, or `user_email` — assert this programmatically, it is the guard against the whole single-user model being bypassed
- `ListResources` returns the 5 URIs; `ListPrompts` returns the 3 names
- one in-process round trip: `initialize` then `tools/call` on `habits_list`, asserting a well-formed result

Use an in-memory transport for the round trip. Check the SDK for its in-memory or client/server pair helper rather than spinning up an HTTP server.

### Validate

```
mise run lint
mise run test
```

---

## Commit 4 — `feat(api): serve MCP over streamable HTTP and stdio`

### 4A. Config

`apps/api/internal/platform/config/config.go` gains:

```go
MCPEnabled  bool          // MY_MCP_ENABLED, default false
MCPToken    string        // MY_MCP_TOKEN, no default
MCPBind     string        // MY_MCP_BIND, default "127.0.0.1"
MCPReadOnly bool          // MY_MCP_READONLY, default false
```

Follow the existing `secureCookies()` pattern: invalid bool values return an error from `Load()`, never silently default. There is already a `defaultEnv` helper — add a `boolEnv(key string, fallback bool) (bool, error)` alongside it and reuse it rather than repeating `strconv.ParseBool` three times.

Validation, enforced in `Load()`:

- `MCPEnabled` true and `MCPToken` shorter than 32 chars → return error. Refuse to boot rather than serving finance data behind a weak token.
- `MCPBind` set to anything other than `127.0.0.1` or `localhost` → **not** an error, but must produce a warning at startup (see 4B).

`MCPToken` must never appear in any log line, error message, or `--version` output. When `Config` gets printed or logged anywhere, verify the token is not included. Grep for existing config logging before you finish.

### 4B. HTTP transport

In `cmd/api/router.go`, when `cfg.MCPEnabled`:

```go
handler := mcpsdk.NewStreamableHTTPHandler(
    func(r *http.Request) *mcpsdk.Server { return mcpServer },
    &mcpsdk.StreamableHTTPOptions{ /* MaxRequestBodyBytes set explicitly */ },
)
r.Handle("/mcp", bearerAuth(cfg.MCPToken)(handler))
r.Handle("/mcp/*", ...)  // confirm whether the SDK handler needs a subtree route
```

Placement rules:

- `/mcp` mounts at router root, **not** under `/api/v1`. It is a different protocol, not a REST resource.
- It must sit **outside** `RequireAuth` and `CSRFProtect`. Different auth axis. Do not attempt to make cookies work here.
- When `MCPEnabled` is false, the route is not registered at all ⇒ 404. Do not register-and-403.

`bearerAuth` middleware, new file `apps/api/internal/shared/middleware/bearer.go`:

- reads `Authorization: Bearer <token>`
- compares with `crypto/subtle.ConstantTimeCompare`. Note: the existing CSRF middleware uses `strings.EqualFold` and is timing-leaky. Do not copy that pattern here, and do not fix CSRF in this track.
- missing or malformed header → 401 with `WWW-Authenticate: Bearer`
- mismatch → 401, same response as missing. Do not distinguish.
- returns JSON errors consistent with `internal/shared` response helpers

Bind address: `cfg.MCPBind` constrains the listener. Current code is `http.ListenAndServe(":"+cfg.APIPort, r)`, which listens on all interfaces. Changing the main listener's bind address would alter existing behavior, which is out of scope for this commit.

Decision: keep the single listener as-is. Instead, enforce the localhost restriction in the bearer middleware by rejecting `/mcp` requests whose `RemoteAddr` is not loopback, unless `MCPBind` was explicitly widened. Log a startup warning when MCP is enabled with a non-loopback `MCPBind`. Document that reverse-proxying `/mcp` requires setting `MY_MCP_BIND` explicitly.

If you find a cleaner approach — for example a second `http.Server` on its own port bound to `MCPBind` — raise it in your report before implementing. Do not switch approaches unilaterally.

### 4C. stdio transport

New file: `apps/api/cmd/mcp/main.go`

- `--version` flag, same as `cmd/api`
- `--check-update` flag (implemented in Phase 6, stub it here or defer the flag entirely — your call, state which)
- logger writes to **stderr only**. stdout is the MCP protocol channel. A single stray stdout write corrupts the session. Verify `plogger.New()` targets stderr; if it writes to stdout, construct a stderr logger locally in `cmd/mcp` rather than changing `plogger` for everyone.
- `bootstrap.New` → `mcp.NewServer` → `server.Run(ctx, &mcpsdk.StdioTransport{})`
- honours `MY_MCP_READONLY`. Does **not** require `MY_MCP_TOKEN` — stdio has no network surface, so a token adds nothing. Does **not** require `MY_MCP_ENABLED`; running the binary is itself the opt-in.
- context cancelled on SIGINT/SIGTERM, then `app.Close()`

Migration path: this is where the relative `"migrations"` path flagged in Phase 1 must be resolved. A standalone `my-mcp` in `~/.local/bin` will not find `./migrations`. Options: embed migrations with `go:embed`, or have `cmd/mcp` skip migrations entirely and require the dashboard to have run them. **Prefer skipping** — a standalone MCP binary silently migrating a finance database is worse than a clear error telling the user to run `mise run migrate`. If you skip, `bootstrap.New` needs a `SkipMigrations bool` option and `cmd/mcp` must fail with an actionable message when the schema is absent.

### 4D. mise task

`mise.toml` gains:

```toml
[tasks."build:mcp"]
description = "Build standalone MCP server binary"
run = "mkdir -p ../../bin && go build -o ../../bin/my-mcp ./cmd/mcp"
dir = "apps/api"
```

Add it to the `build` task's chain. Note `build:api` depends on `copy-web-assets.sh` having run; `build:mcp` does not embed web assets, so it can run independently.

Update `clean` if needed — it already removes `bin/`.

### 4E. Tests

`cmd/api/router_test.go` additions:

- `MCPEnabled: false` ⇒ `POST /mcp` returns 404
- `MCPEnabled: true`, no `Authorization` header ⇒ 401
- wrong token ⇒ 401
- correct token ⇒ not 401 (a real MCP `initialize` body returning 200 is better if practical)
- non-loopback `RemoteAddr` with default `MCPBind` ⇒ rejected

New `apps/api/internal/shared/middleware/bearer_test.go`: table-driven over missing / malformed / wrong / correct.

New `apps/api/internal/platform/config/config_test.go` additions (file already exists from Track 1):

- `MY_MCP_ENABLED=true` with a short token ⇒ `Load()` errors
- `MY_MCP_ENABLED=true` with a 32+ char token ⇒ ok
- `MY_MCP_BIND` defaults to `127.0.0.1`
- invalid bool in any of the three bool vars ⇒ error

### Validate

```
mise run lint
mise run test
mise run build
```

Manual check — REQUIRED, automated tests will not catch a broken stdio handshake:

```
MY_MCP_ENABLED=true ./bin/my-mcp
```

Paste an `initialize` request on stdin and confirm a clean single-line JSON response with nothing else on stdout. Report the actual output.

---

## Commit 5 — `ci: add goreleaser release pipeline with provenance attestation`

### 5A. Goreleaser config

New file: `.goreleaser.yaml` at repo root.

```yaml
version: 2

before:
  hooks:
    - pnpm install --frozen-lockfile
    - pnpm --filter @my/web build
    - bash scripts/copy-web-assets.sh
    - go -C apps/api mod download

builds:
  - id: my
    dir: apps/api
    main: ./cmd/api
    binary: my
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X github.com/jjspscl/my/internal/platform/version.Version={{.Version}}
      - -X github.com/jjspscl/my/internal/platform/version.Commit={{.ShortCommit}}
      - -X github.com/jjspscl/my/internal/platform/version.Date={{.CommitDate}}

  - id: my-mcp
    dir: apps/api
    main: ./cmd/mcp
    binary: my-mcp
    # same env / goos / goarch / ldflags

archives:
  - id: my
    ids: [my]
    name_template: 'my_{{.Version}}_{{.Os}}_{{.Arch}}'
  - id: my-mcp
    ids: [my-mcp]
    name_template: 'my-mcp_{{.Version}}_{{.Os}}_{{.Arch}}'

checksum:
  name_template: 'checksums.txt'

changelog:
  use: github
  groups:
    - title: Features
      regexp: '^.*?feat(\(.+\))??!?:.+$'
      order: 0
    - title: Fixes
      regexp: '^.*?fix(\(.+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999

release:
  draft: false
  prerelease: auto
```

Two things to verify rather than assume:

1. **Embed ordering.** `apps/api/internal/platform/web/static/` must be populated before goreleaser compiles `cmd/api`, or `go:embed` fails. The `before.hooks` handle this, but confirm hooks run before all builds, not per-build.
2. **`ids` vs `builds` key name.** Goreleaser renamed the archive-to-build reference key across versions. Check the schema for the version of goreleaser actually running in CI and use the correct key. Do not guess.

`modernc.org/sqlite` with `CGO_ENABLED=0` — verify this actually cross-compiles for all four targets. `modernc.org/sqlite` is pure Go so it should, but confirm with a snapshot build before tagging. If a target fails, report it rather than dropping the platform.

Add `dist/` to `.gitignore`.

### 5B. Release workflow

New file: `.github/workflows/release.yml`

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write
  id-token: write
  attestations: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - uses: pnpm/action-setup@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm
      - uses: goreleaser/goreleaser-action@v7
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      - uses: actions/attest-build-provenance@v3
        with:
          subject-checksums: ./dist/checksums.txt
```

`fetch-depth: 0` is required — goreleaser needs full history for the changelog and to resolve the version from the tag.

Match the go-version and node-version to the existing `api-ci.yml` and `web-ci.yml` so all three stay consistent.

Provenance attestation lets an agent run `gh attestation verify` before executing a downloaded binary that has full CRUD on finance data. Do not skip it.

### 5C. mise tasks

```toml
[tasks."release:check"]
description = "Dry-run a release build without publishing"
run = "goreleaser release --snapshot --clean --skip=publish"

[tasks."release:tag"]
description = "Validate, then create an annotated release tag (push is manual)"
```

`release:tag` must: refuse on a dirty working tree, refuse when not on `main`, run `mise run lint test typecheck build`, then create an annotated tag. It must **not** push. Tag pushes trigger real publishes — that stays a deliberate human action.

Add `goreleaser` to `mise.toml` `[tools]` so `release:check` works locally without a manual install.

### 5D. Validate

```
mise run lint
mise run test
mise run release:check
```

`release:check` must produce all 8 binaries (2 binaries × 4 platforms) plus `checksums.txt` in `dist/`. Verify:

```
./dist/my_darwin_arm64/my --version
```

reports a snapshot version, **not** `dev`. If it says `dev`, the ldflags path is wrong — most likely the module path in `-X` does not match. Fix before finishing.

Do not create or push any git tag in this commit. Tagging `v1.0.0` is a separate step after the whole track is reviewed.

---

## Commit 6 — `feat(scripts): add MCP install script with checksum verification`

### 6A. Install script

New file: `scripts/install-mcp.sh`. POSIX `sh`, not bash — it runs via `curl | sh` on unknown machines.

Behavior:

1. detect OS (`darwin`/`linux`) and arch (`amd64`/`arm64`) via `uname -sm`; unsupported combination exits with a clear message listing what is supported
2. resolve the latest tag from `https://api.github.com/repos/jjspscl/my/releases/latest`, or honour a `MY_MCP_VERSION` env override
3. download the matching `my-mcp_<version>_<os>_<arch>` archive and `checksums.txt`
4. verify sha256 against `checksums.txt` — **abort on mismatch**, do not warn-and-continue
5. when `gh` is on PATH, additionally run `gh attestation verify --repo jjspscl/my`; when absent, print that provenance was not verified and continue
6. extract to `~/.local/bin/my-mcp`, `chmod +x`
7. warn when `~/.local/bin` is not on `PATH`
8. print client config snippets and required env vars

Hard constraints:

- **Never write to any agent's config file.** Print the snippet, let the user or agent paste it. Silently editing a global agent config is a surprise with no undo.
- Use a `mktemp -d` working directory and clean it up on exit via `trap`.
- No `sudo`. Ever. If `~/.local/bin` is not writable, fail with instructions.
- Every download over HTTPS. Fail on non-200.

### 6B. Update check

`my-mcp --check-update`:

- compares `version.Version` against the latest tag from the GitHub releases API
- prints current, latest, and the install command when a newer version exists
- prints "up to date" otherwise
- exits 0 in both cases; network failure exits non-zero with a clear message
- prints nothing when `version.Version == "dev"` beyond a note that this is a dev build

Explicitly **not** implementing: self-download, self-replace, auto-update on startup, background version checks. A binary that rewrites its own executable while holding credentials to a finance database is a bad trade for convenience. Do not add it even if it seems helpful.

### 6C. Tests

Shell scripts are awkward to unit test and there is no shell test harness in this repo. Do not add one.

Instead:

- Go test for the version-comparison logic (`--check-update` semver compare), including equal, newer, older, and malformed remote tags
- run `shellcheck scripts/install-mcp.sh` if available and fix what it reports; note in your report whether shellcheck was available
- manually run the script against a locally-served `dist/` from `release:check` if practical; if not practical without a real release, say so plainly rather than claiming it was tested

### Validate

```
mise run lint
mise run test
sh -n scripts/install-mcp.sh   # syntax check at minimum
```

---

## Commit 7 — `docs: document MCP setup, client configs, and release process`

### 7A. New `docs/mcp.md`

Sections:

- what the MCP server exposes, and the single-user constraint
- full tool inventory: 28 tools in a table with one-line descriptions, destructive ones flagged
- resources and prompts
- env vars: `MY_MCP_ENABLED`, `MY_MCP_TOKEN`, `MY_MCP_BIND`, `MY_MCP_READONLY`
- transport comparison: when to use HTTP vs stdio
- per-client config for all five clients, **both** transports where supported:
  - Claude Code — `claude mcp add --transport http my http://127.0.0.1:8080/mcp --header "Authorization: Bearer ..."`
  - Codex — `~/.codex/config.toml`, `[mcp_servers.my]`
  - opencode — `opencode.jsonc` remote MCP entry
  - OpenClaw — `openclaw mcp add my --launch "streamable-http http://127.0.0.1:8080/mcp"`
  - Hermes — bearer token supplied via an OS env var name
- verify the exact flag and key syntax for each client rather than copying the shapes above from memory. They came from search results and may be stale. Where you cannot confirm a client's syntax, say so in the doc instead of guessing.
- release process: tag format, what CI publishes, how to verify provenance
- security section: why the token exists, why localhost by default, what full CRUD means

### 7B. README updates

- new "MCP server for coding agents" section with the one-line install, written so an agent can execute it directly
- state plainly that stdio mode needs direct database and Redis access, so it only works on a machine that already holds your `my` data
- commands table gains `build:mcp`, `release:check`, `release:tag`
- Stack section mentions the MCP server
- fix the existing MCP list at README `:145-155` — it omits `engram`
- link `docs/mcp.md` from the docs index

### 7C. AGENTS.md updates

- Runtime facts: `/mcp` endpoint, `my-mcp` binary, new env vars
- Canonical commands: the three new mise tasks
- MCP inventory: distinguish MCPs that `my` *consumes* during development from the MCP server `my` now *provides*. These are different things and conflating them will confuse future agents.

### 7D. `.env.example`

Under a new `# MCP server` section:

```
# MCP server for coding agents (Codex, Claude Code, opencode, Hermes, OpenClaw)
MY_MCP_ENABLED=false
# Required when MY_MCP_ENABLED=true. Minimum 32 chars. Generate: openssl rand -hex 32
MY_MCP_TOKEN=
# Bind address for /mcp. Widening this exposes full finance CRUD.
MY_MCP_BIND=127.0.0.1
# Set true to register read-only tools only.
MY_MCP_READONLY=false
```

Placeholder only. Never a real token. Do not touch the gitignored local `.env`.

### 7E. Other docs

- `docs/architecture.md` — MCP adapter in the directory map and runtime description
- `docs/backend-ddd.md` — note that `internal/platform/mcp/` is an inbound adapter, and why it is not a bounded context
- `docs/opencode.md` — clarify the consumed-vs-provided MCP distinction
- **Do not** touch `ROADMAP.md`.

### Validate

```
mise run lint
mise run test
mise run typecheck
mise run build
```

Re-read every command and config snippet you wrote and confirm it matches what the code actually does. Docs that drift from reality are worse than no docs.

---

## Out of scope — do not start these

Even where they look adjacent:

- `/api/v1/sync/*` backend
- fixing the timing-leaky `strings.EqualFold` CSRF compare (separate security track)
- OAuth 2.1 for MCP — static bearer was chosen deliberately
- Windows release targets
- MCP tool pagination or cursor support
- a `dashboard` bounded context (the snapshot resource composes existing services instead)
- rate limiting, CORS
- production Docker image or container publishing
- Homebrew tap / package manager distribution
- offline queue idempotency keys
- `ROADMAP.md` refresh
- self-updating binary behavior

---

## Commit workflow

Load the `git-commit-and-push` skill **before the first commit** and follow it for message format, splitting, and validation gates.

- branch `feat/mcp-server` off `main`; never commit to `main` directly
- stage specific files, never `git add .`
- 7 commits as laid out above; do not squash them
- working tree is currently clean, so anything unexpected in `git status` is yours — check before staging
- **never commit a real `MY_MCP_TOKEN`**, in `.env.example`, docs, tests, or fixtures. Test tokens must be obviously fake, e.g. `test-token-0000000000000000000000000000`.
- do not create or push the `v1.0.0` tag. That happens after review.

---

## Report back with

- files created and changed, per commit
- `lint` / `test` / `typecheck` / `build` output per phase
- `mise run release:check` output, plus the `--version` string from a snapshot binary — state whether ldflags injection actually worked
- the stdio `initialize` handshake output from Phase 4, verbatim
- what you decided about the migrations path for `cmd/mcp`, and why
- whether `repo-architect` agreed with `internal/platform/mcp/` placement
- exact goreleaser version used and which archive-reference key name it wanted
- whether `modernc.org/sqlite` cross-compiled cleanly to all four targets
- any client config syntax you could not verify
- whether shellcheck was available, and whether the install script was genuinely tested end to end or only syntax-checked
- anything that turned out different from this handoff's assumptions

---

## Assumptions worth flagging

Stated so you can correct them rather than work around them:

1. Go SDK v1.7.0's API shape is taken from docs, not from reading the module. Verify actual function signatures — `NewStreamableHTTPHandler`, `StreamableHTTPOptions`, `ToolAnnotations`, typed tool registration — before writing against them. Adjust and report if they differ.
2. `database.Open` and `predis.NewClient` return types are assumed to be `*sql.DB` and `*redis.Client`. Confirm in the platform packages.
3. Bootstrap tests are assumed able to reach a test database and Redis. If CI cannot, skip gracefully and report.
4. Goreleaser `before.hooks` are assumed to run once before all builds. If they run per-build, the web asset copy runs redundantly — acceptable, but note it.
5. The five client config syntaxes came from web search on 2026-08-08 and may already be stale.
