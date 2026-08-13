# Finance agent

The finance agent is a deployed end-user agent (Hermes runtime) speaking to
the `my` MCP server. Its profile lives in `agent/finance/` and its skills in
`skills/`.

## Skills

End-user skills live in `skills/` and follow standard Agent Skills
conventions (opencode-compatible frontmatter), so they are not tied to one
agent runtime. Development skills for coding agents stay in
`.opencode/skills/` and are separate.

`finance-core` is the shared base. Load it before any other finance skill.

| Skill | Purpose | Tools |
| --- | --- | --- |
| `finance-core` | Shared conventions: units, dates, currencies, insufficient data | `finance_today_total`, `finance_list_wallets` |
| `spending-review` | Analyze spending, cash flow, savings rate, anomalies | `finance_spending_summary`, `finance_cash_flow_summary`, `finance_category_trend`, `finance_anomalies`, `finance_savings_rate` |
| `budget-planner` | Budget health, plan, and category classification | `finance_budget_health`, `finance_budget_summary`, `finance_upsert_budget`, `finance_classify_category` |
| `financial-goals` | Savings goal progress and management | `finance_goal_health`, `finance_create_goal`, `finance_update_goal`, `finance_add_goal_contribution` |
| `monthly-digest` | Composed monthly summary | `finance_monthly_digest` |
| `bill-audit` | Bill reconciliation, recurring charges, payments | `finance_bill_reconciliation`, `finance_recurring_charges`, `finance_upcoming_bills`, `finance_pay_bill` |
| `affordability` | Purchase runway and emergency fund | `finance_affordability`, `finance_emergency_fund` |

## Conventions

- Tool names are bare `finance_*` names. The Hermes runtime prefixes them as
  `mcp_my_finance_*` (see `docs/mcp.md`).
- Amounts are minor units (cents); months are `YYYY-MM`; timestamps are
  RFC3339. Never sum across currencies.
- Wallet balances are liquid balance, not net worth.
- `sufficient: false` results are not definitive; surface `assumptions` and
  `explanation` verbatim.
- Writes require explicit user confirmation before every call.
- No secrets, tokens, or personal data in profile or skill content.

## Profile

`agent/finance/SOUL.md` defines the agent identity: single user from
`MY_USER_EMAIL`, analyst-not-bank, confirm-before-write (Hermes enforces no
annotation gate), and refusals (net worth framing, cross-currency summing,
bank scraping, trading, tax filing, market data).

`agent/finance/README.md` documents Hermes wiring: Streamable HTTP config,
`mcp_my_` tool prefix, and `MY_MCP_READONLY=true` as the technical write
gate.

## See also

- `docs/mcp.md` — server transports, tools, prompts, client configuration
- `docs/finance-analytics.md` — analytics endpoints and semantics