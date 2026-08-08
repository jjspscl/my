# MCP server

`my` provides Model Context Protocol access for Codex, Claude Code, opencode, Hermes, and OpenClaw.

Server surface is single-user. MCP handlers inject `MY_USER_EMAIL`; clients cannot select another user.

## Install latest `my-mcp`

macOS/Linux, amd64/arm64:

```bash
curl -fsSL https://raw.githubusercontent.com/jjspscl/my/v1.0.1/scripts/install-mcp.sh | sh
```

Installer resolves latest GitHub release unless `MY_MCP_VERSION` is set, verifies the archive SHA-256 against `checksums.txt`, verifies the archive's GitHub build provenance when `gh` exists, installs to `~/.local/bin/my-mcp`, and prints configuration. It does not edit client configuration.

Check updates:

```bash
my-mcp --check-update
```

Output suggests rerunning installer. Binary never replaces itself.

## Transports

### stdio

Use standalone `my-mcp`. It requires direct access to configured database and Redis. It skips migrations; run `mise run migrate` from repo or deploy schema before connecting.

Typical command:

```text
~/.local/bin/my-mcp
```

Set `MY_MCP_READONLY=true` for read-only tool registration.

### Streamable HTTP

Run main dashboard with:

```bash
export MY_MCP_ENABLED=true
export MY_MCP_TOKEN="$(openssl rand -hex 32)"
mise run dev:api
```

Endpoint: `http://127.0.0.1:8081/mcp`.

MCP runs on its own listener, separate from the dashboard. It is **not** served on `MY_API_PORT`; requesting `/mcp` on the dashboard port returns 404. The dedicated listener is what makes `MY_MCP_BIND` a real interface restriction, and binding to loopback is also what activates the SDK's DNS rebinding protection, which rejects requests carrying a non-loopback `Host` header with 403.

Setting `MY_MCP_BIND` to anything other than loopback exposes the full finance read and write surface to the network behind only a static bearer token, disables the rebinding protection, and logs a warning at startup. Do that only behind deliberate network controls or a tunnel.

HTTP uses bearer auth. It does not use the dashboard's session cookie or CSRF protection: those are browser mechanisms and do not fit a CLI client.

Required HTTP variables:

| Variable | Default | Meaning |
|---|---|---|
| `MY_MCP_ENABLED` | `false` | Start the MCP listener; disabled means no listener at all |
| `MY_MCP_TOKEN` | empty | Minimum 32 chars; server refuses to boot otherwise |
| `MY_MCP_BIND` | `127.0.0.1` | Interface the MCP listener binds to |
| `MY_MCP_PORT` | `8081` | Port for the MCP listener |
| `MY_MCP_READONLY` | `false` | Omit all write tools when true |

The server also requires `MY_USER_EMAIL`, `MY_DATABASE_URL`, and `MY_REDIS_URL`. `MY_USER_EMAIL` has no default and every tool uses it as the data ownership key, so the server refuses to boot when it is unset.

Never commit the token. Full CRUD can modify or delete finance data.

## Tools

Amounts use minor currency units, dates use RFC3339 unless tool says `YYYY-MM-DD`, and month uses `YYYY-MM`.

### Read tools

| Name | Purpose |
|---|---|
| `finance_list_transactions` | List transactions in date range |
| `finance_today_total` | Today's income and expense totals |
| `finance_budget_summary` | Budget category summary |
| `finance_list_bills` | List recurring bills |
| `finance_upcoming_bills` | Upcoming bill occurrences and payment status |
| `finance_list_goals` | Savings goals and progress |
| `finance_list_wallets` | Wallets with balances |
| `finance_list_transfers` | Wallet transfers |
| `habits_list` | Active habits, status, streak |
| `habits_completions` | Grouped completions by date |

### Write tools

| Name | Purpose |
|---|---|
| `finance_create_transaction` | Create income/expense |
| `finance_delete_transaction` ⚠️ | Permanently delete transaction |
| `finance_upsert_budget` | Replace month budget categories |
| `finance_create_bill` | Create recurring bill |
| `finance_update_bill` | Update recurring bill |
| `finance_delete_bill` ⚠️ | Permanently delete bill |
| `finance_pay_bill` ⚠️ | Write paid payment record |
| `finance_create_goal` | Create savings goal |
| `finance_update_goal` | Update savings goal |
| `finance_delete_goal` ⚠️ | Permanently delete goal |
| `finance_add_goal_contribution` | Add goal contribution, optional transfer |
| `finance_create_wallet` | Create wallet |
| `finance_update_wallet` | Update wallet |
| `finance_archive_wallet` ⚠️ | Archive wallet |
| `finance_create_transfer` | Transfer between wallets |
| `habits_create` | Create habit |
| `habits_toggle` | Toggle completion for date |
| `habits_archive` ⚠️ | Archive habit |

Write tools are omitted from MCP `tools/list` when `MY_MCP_READONLY=true`.

## Resources

- `my://wallets`
- `my://budget/current`
- `my://bills/upcoming`
- `my://habits/today`
- `my://dashboard/snapshot`

All resources return JSON. Dashboard snapshot composes today's total, upcoming bill count, and habit completion ratio.

Clients differ in how they consume resources. Some read them as ambient context; others, including Hermes, expose them as on-demand list and read tools rather than fetching them at startup. Resources are advertised either way.

## Prompts

- `weekly_finance_review`
- `budget_health_check`
- `habit_streak_report`

Prompts guide tool calls; they do not embed personal data.

## Client configuration

Client CLI/config syntax changes across releases. Verify each client's current help output before applying snippets.

### Claude Code HTTP

```bash
claude mcp add --transport http my http://127.0.0.1:8081/mcp \
  --header "Authorization: Bearer $MY_MCP_TOKEN"
```

### Codex stdio

```toml
[mcp_servers.my]
command = "/Users/you/.local/bin/my-mcp"
```

### opencode stdio

```jsonc
{
  "mcp": {
    "my": {
      "type": "local",
      "command": ["/Users/you/.local/bin/my-mcp"]
    }
  }
}
```

### OpenClaw HTTP

```bash
openclaw mcp add my --launch "streamable-http http://127.0.0.1:8081/mcp"
```

Configure bearer token through OpenClaw's environment/header mechanism. Do not put token in shell history or committed config.

### Hermes

Config lives at `~/.hermes/config.yaml`:

```yaml
mcp_servers:
  my:
    url: "http://127.0.0.1:8081/mcp"
    headers:
      Authorization: "Bearer ${MY_MCP_TOKEN}"
    connect_timeout: 60
    timeout: 120
    sampling:
      enabled: false
```

`${MY_MCP_TOKEN}` resolves from `~/.hermes/.env`, then system environment, then defaults. Put the same value the server uses in `~/.hermes/.env`:

```
MY_MCP_TOKEN=<same value as the API's MY_MCP_TOKEN>
```

Hermes prefixes tools as `mcp_my_<tool>`, so `finance_list_transactions` is called as `mcp_my_finance_list_transactions`.

`sampling: {enabled: false}` is set because this server never issues `sampling/createMessage` requests. Leaving it enabled grants a capability nothing uses.

Resources and prompts are available on demand rather than injected at startup: Hermes registers `mcp_my_list_resources`, `mcp_my_read_resource`, `mcp_my_list_prompts`, and `mcp_my_get_prompt` when the server advertises those capabilities.

Hermes does not enforce `destructiveHint` or `readOnlyHint`. An agent may choose to confirm before destructive calls, but the client applies no technical gate. Run the server with `MY_MCP_READONLY=true` if you want write tools to be genuinely unavailable rather than merely discouraged.

For stdio instead, note that Hermes strips the subprocess environment to `PATH`, `HOME`, `USER`, `LANG`, `LC_ALL`, `TERM`, `SHELL`, `TMPDIR`, and `XDG_*`. Anything else must be listed explicitly under `env` or it is dropped:

```yaml
mcp_servers:
  my:
    command: "/Users/<you>/.local/bin/my-mcp"
    env:
      MY_DATABASE_URL: "${MY_DATABASE_URL}"
      MY_REDIS_URL: "${MY_REDIS_URL}"
      MY_USER_EMAIL: "you@example.com"
      MY_MCP_READONLY: "true"
    connect_timeout: 60
    timeout: 120
```

Omitting any of those three causes the binary to exit at startup rather than fall back to defaults.

## Releases

Release trigger: annotated SemVer tag matching `v*`, starting at `v1.0.0`.

GoReleaser publishes:

- `my` dashboard binary
- `my-mcp` standalone MCP binary
- Linux/macOS amd64/arm64 archives
- `checksums.txt`
- GitHub build provenance attestation

Local snapshot:

```bash
mise run release:check
```

Create tag after review:

```bash
MY_RELEASE_VERSION=v1.0.0 mise run release:tag
```

Tag push runs `.github/workflows/release.yml`. The workflow publishes the release and attests build provenance for every archive listed in `checksums.txt`.

Verify a downloaded archive manually:

```bash
gh attestation verify my-mcp_1.0.1_darwin_arm64.tar.gz --repo jjspscl/my
```

Note that `subject-checksums` attests the artifacts enumerated in the checksum file, not the checksum file itself. Verify an archive, not `checksums.txt`.

Do not push a tag until snapshot and full validation pass.

## Security

- MCP disabled by default.
- HTTP token minimum length: 32 characters.
- HTTP defaults to a dedicated loopback-bound listener at `127.0.0.1:8081`.
- `MY_USER_EMAIL` is required; server refuses to boot when unset.
- Bearer comparison uses constant-time comparison.
- Session cookie and CSRF middleware do not protect `/mcp`; bearer token does.
- Tool audit logs include name, duration, outcome, and error text only. Arguments/results are never logged.
- Destructive tools are annotated and described as irreversible.
- Installer verifies the archive checksum, and the archive's build provenance when `gh` is available.
- Do not expose HTTP endpoint publicly without adding network controls and rotating token on suspected disclosure.
