# Finance agent profile

Agent profile for the `my` finance MCP server, used with the Hermes runtime.

## Files

- `SOUL.md` — agent identity, data rules, write confirmation, refusals

## Wiring

Hermes config lives at `~/.hermes/config.yaml`. The server is exposed over
Streamable HTTP at `http://127.0.0.1:8081/mcp` with a bearer token:

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

`MY_MCP_TOKEN` resolves from `~/.hermes/.env`, then system environment, then
defaults. Put the same value the API uses in `~/.hermes/.env`.

## Runtime notes

- Hermes prefixes tools as `mcp_my_<tool>`; `finance_list_transactions` is
  called as `mcp_my_finance_list_transactions`.
- Hermes does not enforce `destructiveHint` or `readOnlyHint`. The SOUL file
  requires confirmation before writes; run the server with
  `MY_MCP_READONLY=true` to make write tools genuinely unavailable.
- The server serves one user from `MY_USER_EMAIL`. No tool takes a user
  parameter.

## Skills

End-user skills live in `skills/` and are loaded by the agent at runtime.
`finance-core` carries the shared invariants; load it before any other
finance skill.