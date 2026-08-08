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

Endpoint: `http://127.0.0.1:8080/mcp`.

Default bind policy rejects non-loopback clients. `MY_MCP_BIND=0.0.0.0` permits remote clients and logs warning. Use only behind network controls or tunnel. HTTP uses bearer auth and does not use dashboard cookie/CSRF auth.

Required HTTP variables:

| Variable | Default | Meaning |
|---|---|---|
| `MY_MCP_ENABLED` | `false` | Register `/mcp`; disabled returns 404 |
| `MY_MCP_TOKEN` | empty | Minimum 32 chars when HTTP enabled |
| `MY_MCP_BIND` | `127.0.0.1` | Local-only policy unless widened |
| `MY_MCP_READONLY` | `false` | Omit all write tools when true |

Never commit token. Full CRUD can modify or delete finance data.

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

## Prompts

- `weekly_finance_review`
- `budget_health_check`
- `habit_streak_report`

Prompts guide tool calls; they do not embed personal data.

## Client configuration

Client CLI/config syntax changes across releases. Verify each client's current help output before applying snippets.

### Claude Code HTTP

```bash
claude mcp add --transport http my http://127.0.0.1:8080/mcp \
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
openclaw mcp add my --launch "streamable-http http://127.0.0.1:8080/mcp"
```

Configure bearer token through OpenClaw's environment/header mechanism. Do not put token in shell history or committed config.

### Hermes

Configure custom MCP server with transport `Streamable HTTP`, URL `http://127.0.0.1:8080/mcp`, and bearer token sourced from an environment variable. Hermes' UI/config names may change; confirm current client instructions.

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
- HTTP defaults to loopback-only remote policy.
- Bearer comparison uses constant-time comparison.
- Session cookie and CSRF middleware do not protect `/mcp`; bearer token does.
- Tool audit logs include name, duration, outcome, and error text only. Arguments/results are never logged.
- Destructive tools are annotated and described as irreversible.
- Installer verifies the archive checksum, and the archive's build provenance when `gh` is available.
- Do not expose HTTP endpoint publicly without adding network controls and rotating token on suspected disclosure.
