# HANDOFF — v1.0.2: fix phantom tenant + Hermes docs

Status: **implemented and released**. v1.0.2 published; tree clean and synced with origin.

Goal: fix unvalidated `MY_USER_EMAIL` creating a phantom tenant via MCP, correct Hermes config docs and resource claim, release v1.0.2.

Branch: `main`. Latest published tag `v1.0.2`.

---

## Context

User asked for a prompt to give Hermes for a compatibility dry-run. Hermes replied with a detailed assessment that corrected me on one claim (resources/prompts are available on-demand, not ignored) and listed one thing it couldn't verify: what happens when `MY_USER_EMAIL` is missing.

That question led to finding a real bug: `config.Load()` reads `MY_USER_EMAIL` with no default and no validation. Unset means empty string, bootstrap succeeds, MCP server starts, and all 28 tools pass `""` as the ownership key. Read tools return empty results that look like "no data" rather than "misconfigured." Write tools persist rows owned by the empty string. Ownership checks compare against it consistently, so it forms a coherent phantom tenant.

Only MCP is exposed. The HTTP API is accidentally protected because `auth_service.go:36` compares login email against `config.UserEmail`, and empty can't match a real magic-link login. MCP trusts the config value directly.

Contrast: `MY_MCP_TOKEN` correctly refuses to boot when <32 chars in the same `Load()` function. Validation rigor was inconsistent.

User approved a global fix: `Load()` errors on empty `MY_USER_EMAIL` (affecting `cmd/api`, `cmd/mcp`, and `cmd/migrate`), plus a defensive panic in `NewServer` to guard the adapter independently. Chosen panic over error return because this is a startup misconfiguration that `config.Load()` already rejects, and panic avoids signature churn at the two call sites.

Blast radius: 4 existing `config` tests and 3 MCP test call sites construct configs without `UserEmail` and need the var set. `cmd/migrate` will now require `MY_USER_EMAIL`.

User also approved folding Hermes's verified config into `docs/mcp.md` (replacing the placeholder section), correcting my wrong claim about resources being "dead weight," pinning install URLs to v1.0.2, and releasing immediately.

## Locked decisions

| Axis | Decision |
|---|---|
| Scope | Global guard in `config.Load()` when `MY_USER_EMAIL` is empty; defensive panic in `NewServer`; tests; docs; release v1.0.2 |
| Adapter guard style | Panic in `NewServer`, not error return |
| Hermes config source | Verified by Hermes itself from its own docs; independently corroborated by web search |
| Release scope | Fix + docs, commit, tag v1.0.2, publish |
| Out of scope | Reverse-proxy trust bug (separate finding); timing-leaky CSRF compare; Node 20 CI warnings; no changes to v1.0.0/v1.0.1 |

---

## Implementation plan

### Commit 1: `fix(api): reject empty MY_USER_EMAIL`

**Files:**

`apps/api/internal/platform/config/config.go`
- In `Load()`, read `MY_USER_EMAIL` into a local var before building `Config`
- Return `fmt.Errorf("MY_USER_EMAIL is required")` when empty
- Place check near the existing `MY_MCP_TOKEN` validation (lines 45–48) for consistency

`apps/api/internal/platform/mcp/server.go`
- In `NewServer`, add a guard at the top:
  ```go
  if app.Cfg.UserEmail == "" {
      panic("mcp: NewServer called with empty UserEmail")
  }
  ```
- Place it before the `mcpsdk.NewServer` call
- While here: fix the cosmetic dead parameter. `appVersion` at line 41 is
  `func appVersion(app *bootstrap.App) string { return platformversion.Version }`
  — it takes `app` and never reads it. Either drop the parameter and call
  `platformversion.Version` directly at line 25, or rename to `_`. Prefer
  dropping the helper entirely since it now adds nothing.
- Note: `appVersion` returns `platformversion.Version` (bare version), while
  `cmd/api --version` prints `platformversion.String()` (version + commit +
  date). That asymmetry is intentional — MCP `Implementation.Version` should be
  the bare semver. Leave it.

`apps/api/internal/platform/config/config_test.go`
- New test: `TestLoadRequiresUserEmail` — unset `MY_USER_EMAIL`, call `Load()`, assert error contains "MY_USER_EMAIL is required"
- Fix the existing tests that expect `Load()` to succeed. Each needs
  `t.Setenv("MY_USER_EMAIL", "test@example.com")` before the `Load()` call:
  - `TestLoadSecureCookiesExplicitValue` (line 5, calls `Load()` at 18)
  - `TestLoadSecureCookiesDerivesFromWebURL` (line 36, calls `Load()` at 49)
  - `TestLoadMCPConfig` (line 60, calls `Load()` at 66)
- These three currently assert success and will fail once the guard lands.
- The three negative tests (`TestLoadSecureCookiesRejectsInvalidValue` line 29,
  `TestLoadMCPRejectsShortToken` line 75, `TestLoadMCPRejectsInvalidBoolean`
  line 83) assert only that an error is returned, so they pass either way. Set
  `MY_USER_EMAIL` in them anyway so each test fails for the reason it names
  rather than for a missing user email.

`apps/api/internal/platform/mcp/server_test.go`
- Update 3 call sites constructing `&config.Config{}`, all of which will panic
  once the guard lands:
  - `TestToolRegistry` (func at line 29, `NewServer` call at line 30)
  - `TestReadOnlyToolRegistry` (func at line 53, call at line 54)
  - `TestToolAnnotations` (func at line 61, call at line 62)
- Change each to `&config.Config{UserEmail: "test@example.com"}`
- New test: `TestNewServerPanicsOnEmptyUserEmail` — construct `&bootstrap.App{Cfg: &config.Config{}}`, call `NewServer`, assert panic with `recover()`

**Validation:**
- `go vet ./...` clean
- `go test ./...` passes, including the 5 new/updated tests
- Live check: unset `MY_USER_EMAIL`, try to start `bin/my` → exits with "MY_USER_EMAIL is required" before listening
- Live check: set `MY_USER_EMAIL`, start `bin/my`, verify `/mcp` still serves `tools/list`

**Commit message pattern:**
```
🐛 fix(api): reject empty MY_USER_EMAIL

## Overview
Fail loudly when MY_USER_EMAIL is unset instead of silently starting with
a phantom tenant.

## Why
config.Load() read MY_USER_EMAIL with no default and no validation. Unset
meant empty string, Load() succeeded, and the MCP server started. All 28
tools then passed "" as the ownership key. Read tools returned empty results
that looked like "no data" rather than "misconfigured." Write tools persisted
rows owned by the empty string. Ownership checks compared against it
consistently, so it behaved as a coherent phantom tenant.

Only MCP was exposed. The HTTP API is accidentally protected because
auth_service.go:36 compares login email against config.UserEmail, and empty
can't match a real magic-link login. MCP trusts the config value directly.

Contrast: MY_MCP_TOKEN correctly refuses to boot when <32 chars in the same
Load() function. Validation rigor was inconsistent.

## Changes
- config.Load() returns an error when MY_USER_EMAIL is empty
- NewServer panics on empty UserEmail as a defensive guard (panic chosen over
  error return because config.Load() already rejects it and panic avoids
  signature churn at the two call sites)
- fix cosmetic: appVersion now uses its argument
- tests updated: 4 config tests and 3 mcp tests set MY_USER_EMAIL
- new tests: Load() error on empty, NewServer panic on empty

## Breaking Changes
- cmd/api, cmd/mcp, and cmd/migrate now require MY_USER_EMAIL. Previously
  cmd/migrate inherited an unvalidated empty default.
- Any deployment relying on the implicit empty default will fail at boot with
  "MY_USER_EMAIL is required."

## Validation
- go vet ./..., go test ./... pass
- live: unset var → boot fails; set var → /mcp serves tools/list
```

---

### Commit 2: `docs: correct Hermes config and resource consumption`

**Files:**

`docs/mcp.md`

Replace the placeholder Hermes section (currently 3 lines starting at line 165) with:

```markdown
### Hermes

Hermes connects via Streamable HTTP and exposes tools, resources, and prompts on demand.

**Config:** `~/.hermes/config.yaml`

```yaml
mcp_servers:
  my:
    url: "http://127.0.0.1:8080/mcp"
    headers:
      Authorization: "Bearer ${MY_MCP_TOKEN}"
    connect_timeout: 60
    timeout: 120
    sampling:
      enabled: false
```

**Token resolution:** `${MY_MCP_TOKEN}` resolves via `~/.hermes/.env` → system environment → defaults.

Create `~/.hermes/.env`:

```
MY_MCP_TOKEN=<same value used by the my API's MY_MCP_TOKEN>
```

**Tool naming:** Tools register as `mcp_my_<tool>`. For example, `finance_list_transactions` becomes `mcp_my_finance_list_transactions`.

**Annotations:** Hermes does not enforce `destructiveHint` or `readOnlyHint`. The agent may ask before destructive calls as a policy, but the client itself does not block them. For technical protection, run the server with `MY_MCP_READONLY=true`, which omits write tools from registration entirely.

**stdio transport (not recommended):** Hermes supports stdio via the standalone `my-mcp` binary, but this requires direct database and Redis access:

```yaml
mcp_servers:
  my:
    command: "/Users/<your-macOS-username>/.local/bin/my-mcp"
    env:
      MY_DATABASE_URL: "${MY_DATABASE_URL}"
      MY_REDIS_URL: "${MY_REDIS_URL}"
      MY_USER_EMAIL: "you@example.com"
      MY_MCP_READONLY: "true"
    connect_timeout: 60
    timeout: 120
```

**Environment stripping:** Hermes strips the subprocess environment, inheriting only `PATH`, `HOME`, `USER`, `LANG`, `LC_ALL`, `TERM`, `SHELL`, `TMPDIR`, and `XDG_*`. Anything not explicitly listed under `env` is dropped. Omitting any of the three `MY_*` vars causes the binary to fail with a config error.

**Resources and prompts:** Hermes registers `mcp_my_list_resources`, `mcp_my_read_resource`, `mcp_my_list_prompts`, and `mcp_my_get_prompt` when the server advertises those capabilities. They are not automatically fetched at startup but are available on demand.
```

**Also update:**

- Line 15 (install overview): add "Requires `MY_USER_EMAIL` alongside database/Redis." after the existing env list
- Resources section (starts line 108): add a note that some clients expose resources on demand rather than auto-injecting them as context
- Security section (starts line 205): add "`MY_USER_EMAIL` is required; the server refuses to boot when unset."
- Env table (starts line 57): the table lists only the four `MY_MCP_*` vars. Add a `MY_USER_EMAIL` row, or a sentence above it noting the server also requires `MY_USER_EMAIL`, `MY_DATABASE_URL`, and `MY_REDIS_URL`.

`.env.example`

`MY_USER_EMAIL=you@example.com` is at line 19. Add a comment line above it:

```bash
# Required; no default. Every MCP tool uses this as the data ownership key.
```

**Validation:**
- `mise run lint` (frontend may still have 18 react-refresh warnings; ignore)
- Markdown renders correctly in a viewer
- All URLs, variable names, and paths match the implementation

**Commit message pattern:**
```
📝 docs: correct Hermes config and resource consumption

## Overview
Replace placeholder Hermes section with verified config from Hermes itself,
and correct the claim that resources/prompts are ignored.

## Why
The docs carried a 2-line placeholder referencing Streamable HTTP and bearer
auth but no actual config. After Hermes provided a detailed compatibility
assessment with exact YAML, token resolution, tool prefixing, annotation
non-enforcement, and stdio env stripping, the docs now reflect what an agent
actually needs.

Also: I incorrectly told the user that resources and prompts were "dead
weight" for Hermes because the skill doc says startup discovery calls
list_tools(). Hermes corrected me: it registers mcp_my_list_resources,
mcp_my_read_resource, mcp_my_list_prompts, and mcp_my_get_prompt when the
server advertises those capabilities. On-demand, not auto-injected, but
available. I mistook a narrow implementation note for a capability ceiling.

## Changes
- docs/mcp.md: full Hermes HTTP config with url, headers, timeout,
  connect_timeout, sampling: {enabled: false}
- document token resolution via ~/.hermes/.env
- document mcp_my_* tool prefix
- state that Hermes does not enforce destructiveHint, so MY_MCP_READONLY=true
  is the real gate
- document stdio env stripping and the three required vars
- add resources/prompts on-demand note
- add MY_USER_EMAIL to required-variables list and security section
- .env.example: note that MY_USER_EMAIL is required with no default

## Breaking Changes
- None. Documentation only.

## Validation
- markdown renders correctly
- all variable names, paths, and URLs match implementation
```

---

### Commit 3: `docs: pin install to v1.0.2`

**Files:**

`README.md`
- Line 153: `v1.0.1` → `v1.0.2`

`docs/mcp.md`
- Line 12: `v1.0.1` → `v1.0.2`
- Line 198 (manual attestation example): `my-mcp_1.0.1_` → `my-mcp_1.0.2_`

**Validation:**
- `grep -rn 'v1\.0\.[01]' README.md docs/mcp.md` returns nothing

**Commit message pattern:**
```
📝 docs: pin install to v1.0.2

## Overview
Update pinned install URLs and manual verification example to v1.0.2.

## Why
v1.0.1 shipped without MY_USER_EMAIL validation. The fix lives in the next
tag. The install URL is intentionally pinned to a tag rather than a branch so
documented commands track a stable artifact.

## Changes
- README.md and docs/mcp.md: v1.0.1 → v1.0.2
- manual attestation example: my-mcp_1.0.1_ → my-mcp_1.0.2_

## Breaking Changes
- None. Documentation only.

## Validation
- verified no remaining v1.0.0 or v1.0.1 references in install paths
```

---

### Release v1.0.2

1. Full validation suite:
   ```bash
   mise run lint
   mise run test
   mise run typecheck
   mise run build
   ```

2. Tag:
   ```bash
   MY_RELEASE_VERSION=v1.0.2 mise run release:tag
   ```

3. Push:
   ```bash
   git push origin main
   git push origin v1.0.2
   ```

4. Watch:
   ```bash
   gh run list --repo jjspscl/my --workflow=Release --limit 1
   gh run watch <run-id> --repo jjspscl/my --exit-status
   ```

5. Verify assets:
   ```bash
   gh release view v1.0.2 --repo jjspscl/my --json tagName,assets | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["tagName"],"—",len(d["assets"]),"assets"); [print(" ",a["name"],a["size"]) for a in d["assets"]]'
   ```

6. End-to-end installer test (same as v1.0.1):
   ```bash
   inst=$(mktemp -d)
   curl -fsSL https://raw.githubusercontent.com/jjspscl/my/v1.0.2/scripts/install-mcp.sh -o "$inst/i.sh"
   MY_MCP_INSTALL_DIR="$inst/bin" sh "$inst/i.sh"
   "$inst/bin/my-mcp" --version
   "$inst/bin/my-mcp" --check-update
   rm -rf "$inst"
   ```

7. Verify the fixed guard:
   ```bash
   tmp=$(mktemp -d)
   env MY_API_PORT=8099 MY_DATABASE_URL="file:$tmp/test.db" MY_REDIS_URL=redis://localhost:6379 \
       MY_MCP_ENABLED=true MY_MCP_TOKEN=$(openssl rand -hex 32) \
       bin/my 2>&1 | head -5
   # Should exit with "MY_USER_EMAIL is required" before listening
   rm -rf "$tmp"
   ```

8. Verify the working case still works:
   ```bash
   tmp=$(mktemp -d); TOKEN=$(openssl rand -hex 32)
   env MY_API_PORT=8099 MY_DATABASE_URL="file:$tmp/ok.db" MY_REDIS_URL=redis://localhost:6379 \
       MY_USER_EMAIL=test@example.com MY_MCP_ENABLED=true MY_MCP_TOKEN="$TOKEN" \
       bin/my > "$tmp/log" 2>&1 & pid=$!
   sleep 3
   curl -s -X POST http://127.0.0.1:8099/mcp -H "Authorization: Bearer $TOKEN" \
     -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
     -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}' \
     | python3 -c 'import json,sys; print("tools:",len(json.load(sys.stdin)["result"]["capabilities"]["tools"]))'
   kill $pid 2>/dev/null; rm -rf "$tmp"
   # Should print "tools: <count>"
   ```

Expected: 9 assets (8 archives + checksums.txt), installer succeeds, `--version` reports `1.0.2 (...)`, `--check-update` reports up to date, guard test exits with the required error, working case initializes and lists tools.

---

## Deferred / Out of scope

From the prior production-readiness review (#37 in memory):

1. **Reverse-proxy trust bug:** `RequireBearerToken` checks `r.RemoteAddr` for loopback. Behind any reverse proxy this is always loopback, so the local-only policy silently passes for remote clients. The protection is weaker than stated in docs.
2. **`MY_MCP_BIND` does not bind:** it only toggles the check and a warning. `/mcp` shares the dashboard listener. The name misleads.
3. Rate limiting, graceful shutdown, Slowloris timeouts, CSRF timing leak — all separate tracks.

None block v1.0.2. User chose not to fold them into this release.

---

## Notes for the implementer

- The cosmetic `appVersion` fix is bundled into commit 1 since it's in the same file as the `NewServer` guard. One-line change.
- `cmd/migrate` gaining a `MY_USER_EMAIL` requirement is a side effect worth stating in the commit body but not a primary motivation.
- Hermes's config syntax was verified by Hermes itself from its own docs, then independently corroborated by web search results showing the same schema.
- The stdio env-stripping detail is Hermes-specific and matters because omitting a required var causes a silent drop, not an error at Hermes's config-parse time.
- `sampling: {enabled: false}` is safe because my server never issues `sampling/createMessage` requests. Hermes confirmed this is appropriate.
- Install URL stays pinned to a tag (not `main`) so `curl | sh` executes stable, reviewed code rather than HEAD.

## Success criteria

- `go vet ./...` clean
- `go test ./...` passes
- `mise run lint test typecheck build` all pass
- v1.0.2 publishes 9 assets
- Documented installer succeeds and `--version` reports `1.0.2`
- `MY_USER_EMAIL` unset → boot fails with "required" error
- `MY_USER_EMAIL` set → `/mcp` serves `initialize` and `tools/list`
- No regression in HTTP API or migrations

## Completed result

- ✅ `MY_USER_EMAIL` empty now fails before listener bind.
- ✅ MCP binds dedicated `MY_MCP_BIND:MY_MCP_PORT`, default `127.0.0.1:8081`.
- ✅ SDK DNS-rebinding protection active on loopback listener.
- ✅ Dashboard `/mcp` returns 404; MCP endpoint works only on dedicated port.
- ✅ SIGTERM gracefully shuts down dashboard and MCP servers.
- ✅ Hermes config documents on-demand resources/prompts and server-side read-only gate.
- ✅ v1.0.2 published with 9 assets; live pinned installer verifies checksum and provenance.
