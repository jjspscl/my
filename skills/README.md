# Finance agent skills

End-user skills for the finance agent. Each skill is a standard Agent Skills
`SKILL.md` (opencode-compatible frontmatter) so they are not tied to one agent
runtime.

These are runtime skills for the deployed finance agent. Development skills
live in `.opencode/skills/` and stay separate.

## Skills

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

- Tool names are bare `finance_*` names. A runtime may prefix them (for
  example Hermes exposes them as `mcp_my_finance_*`); the skill text always
  uses the bare name.
- All skills inherit `finance-core` invariants. Load `finance-core` first.
- Writes are never automatic: confirm before creating, updating, deleting, or
  paying anything.
- Never present `sufficient: false` results as definitive. Surface
  `assumptions` and `explanation` fields verbatim.