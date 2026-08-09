# HANDOFF — agent-native finance: analytics, semantic MCP, finance skills

Status: **complete**. Branch `v1.2.0`; Phase 3 (derived analytics) backend complete across `730aed6` (anomalies), `e1724e2` (recurring + bill reconciliation), `093c80b` (emergency fund + affordability + digest + docs). Phase 4 (MCP surface) complete across `e19c756` (server split), `f4afc28` (pay-bill fix + `create_transaction`), `ad9c5c3` (12 analytics tools + prompts), `462de02` (classify tool + docs). Phase 5 (skills, agent profile, dashboard, docs) complete across `b818e6c` (skills), `c359c28` (agent profile), `aaaf7df` (docs), `6f269af` (analytics data layer), `fced594` (finance layout + child routes), `bac6883` (analytics overview).

Goal: turn the finance bounded context into a trustworthy analytics layer, expose it as a small semantic MCP surface, and layer end-user finance agent skills on top — without becoming a bank.

---

## Context

An external proposal document asked for a broad agent-native finance system: a deterministic analytics layer, ~10 new MCP tools, 7 end-user agent skills, a Hermes finance profile, and a dashboard rework. Its direction was right; its sequencing was not, and several of its premises did not match this codebase.

What the proposal got right: analytics belong in the application layer, not in MCP handlers; deterministic aggregation beats making an LLM sum transaction dumps; semantic tools beat CRUD dumps; no gamified health score; no bank credential storage or scraping; development skills stay separate from end-user runtime skills.

What it got wrong, verified against code:

- Foundation defects were slated for its final phase, but every analytic depends on them. They move to Phase 0.
- It treated "transfers must not count as income or expense" as work. Transfers live in their own `wallet_transfers` table and never touch `transactions`, so that invariant already holds structurally. It reduces to adding tests.
- It warned about floating-point money. Money is already `int64` minor units throughout.
- It asked to "preserve multi-currency support." There is nothing to preserve. Wallets carry a `currency` column that nothing enforces or aggregates by, and `transaction_service.go` overwrites it with a global config value.
- Its emergency-fund section required essential-versus-discretionary category classification while its category section forbade migrating. Contradiction resolved below.
- It deferred duplicate-write protection, though the offline mutation queue already replays mutations against write paths that have no idempotency keys.

Defects found during the audit, all pre-existing:

| Defect | Location | Effect |
|---|---|---|
| Transaction currency taken from global config, not the resolved wallet | `application/transaction_service.go:65` | A USD wallet produces PHP-labeled transactions |
| Mixed-currency `SUM`, then currency guessed from an arbitrary row via `LIMIT 1` | `infrastructure/transaction_repo_libsql.go` | Today total is meaningless once a second currency exists |
| No timezone handling anywhere in Go | `transaction_service.go:109` uses UTC; `interfaces/http/finance_handler.go:83` uses server-local | In UTC+8, an expense logged before 08:00 lands on the previous UTC day and disappears from today total |
| `strftime('%Y-%m', transaction_date) = ?` | `infrastructure/budget_repo_libsql.go` | Function on the column defeats `idx_transactions_user_date`; full scan |
| No index on `transactions.wallet_id` | migration `007_wallets.sql` | Four correlated balance subqueries scan |
| N+1 per goal | `application/goal_service.go` `ListSummaries` | One query per goal for current amount |
| N+1 per bill occurrence, plus an O(n²) bubble sort | `application/bill_service.go` `GetUpcoming` | ~90 occurrences per bill, one `FindPayment` each |
| Correct SQL exists but is dead code | `infrastructure/bill_repo_libsql.go` `ListUpcomingBills` | Never called by any service |
| Two writes, no transaction | `application/goal_service.go` `AddContribution` | Failure between the transfer and the contribution leaves an orphan transfer that already moved a wallet balance |
| No idempotency keys on transaction, transfer, or goal-contribution writes | services + migrations | `apps/web/src/shared/sync/mutation-queue.ts` replays mutations; retries duplicate rows |
| `MarkPaid` writes no transaction | `application/bill_service.go` | A bill paid without a transaction is invisible to expense and budget analytics |
| Deleting a linked transaction leaves the bill paid | `ON DELETE SET NULL` in `005_recurring_bills.sql` | Silent divergence between bills and actuals |
| `finance_pay_bill` annotated destructive | `platform/mcp/server.go` | It is an idempotent upsert on `UNIQUE(bill_id, due_date)` |

Only `budget_categories` upsert uses `BeginTx` today. Finance tests are unit-only with in-memory fakes: no repository/SQL, HTTP, timezone, currency, transfer-service, or wallet-balance coverage.

Research consulted: Firefly III documentation on transactions, budgets, categories, subscriptions, and recurring transactions; Firefly III discussion #12057 (actual versus expected bill amounts); Actual Budget category groups. Takeaway: no mainstream open-source finance system hard-codes an essential/discretionary taxonomy, and bills are modeled as expectations linked to transactions, never as substitutes for them.

## Locked decisions

| Axis | Decision |
|---|---|
| Currency | Flexible multi-currency, no FX rate table. The wallet is the currency authority; transaction currency derives from the resolved wallet. `MY_DEFAULT_CURRENCY` demotes to new-wallet default plus reporting base. Add `currency` to budgets, bills, goals |
| Cross-currency transfers | Dual amounts `from_amount_cents` / `to_amount_cents`; user supplies both legs; effective rate implied and captured at transfer time |
| Mixed-currency aggregation | Group by currency, never sum across. Without rates, refuse explicitly: `Cannot aggregate PHP and USD without conversion rates.` Never assume 1:1 |
| Bills | Bills stay expectations, transactions stay facts. `MarkPaid` must not silently create a transaction |
| Bill reconciliation | Optional `create_transaction` flag on pay-bill, default false, doing both in one `BeginTx`. Analytics expose expected versus actual, variance, and paid-without-transaction count as first-class fields. Deleting a linked transaction reverts its payment to `pending` |
| Categories | Additive unscoped `finance_categories(name PK, classification, essential, active, timestamps)`. No foreign key on `transactions.category` |
| Unclassified handling | Any tool depending on essential/discretionary returns `unclassified_share` and refuses with `insufficient_classification` above 20%, listing top unclassified categories |
| Single-user | `MY_USER_EMAIL` is required at boot. New tables omit `user_email`; idempotency key is unique on its own; timezone is one boot-time config; analytics services read identity from config rather than accepting a caller parameter |
| Skills layout | `skills/` for end-user finance skills, `.opencode/skills/` stays development-only, distinction documented |
| Commits | Commit at each phase boundary on `feat/finance-analytics` |
| Out of scope | Assets/liabilities net-worth model (design document only; wallet sums stay labeled "liquid balance"); FX rate table for consolidated base-currency reporting; bank aggregation, scraping, trading, tax filing; market data feeds |

Expectation to state plainly: Phase 0 is roughly a third of total effort and ships zero visible features.

---

## Phase 0 — foundation (blocking)

No new features. Everything downstream is only trustworthy because of this phase. Migrations start at `009`; never rewrite `001`–`008`.

1. **Timezone.** Add `MY_TIMEZONE`, default `Asia/Manila`, resolved once in `config.Load()` into a `*time.Location` on `Config`. Introduce a single date-resolution helper and route every "today", month boundary, and range predicate through it. Remove `time.Now().UTC()` from `transaction_service.go:109`, the server-local `time.Now()` from `finance_handler.go:83`, and `toISOString().split('T')[0]` from the frontend. Tests at 00:00 and 23:59 Manila across month and year boundaries.
2. **Currency correctness.** Derive transaction currency from the resolved wallet in `TransactionService.Create`. Migrations adding `currency` to budgets, bills, goals. Transfer dual-amount migration. Delete the `LIMIT 1` currency guess. Per-currency grouping in every aggregate.
3. **Idempotency.** Client-supplied key on transaction, transfer, and goal-contribution writes, with a unique constraint; replay returns the original record rather than erroring. Wire `apps/web/src/shared/sync/mutation-queue.ts` and the MCP write tools to generate a key once and reuse it across retries.
4. **Atomicity.** `BeginTx` around goal contribution (transfer plus contribution) and around bill payment.
5. **Query shape.** Half-open range predicate replacing `strftime`. Index on `transactions.wallet_id`. Fix the `ListSummaries` N+1. Fix `GetUpcoming` by calling the existing `ListUpcomingBills` SQL. Replace the bubble sort with `sort.Slice`.
6. **Integration harness.** Repository-level tests against a real temporary libSQL file. None exist today, and every later aggregate needs them.

## Phase 1 — category metadata

Migration creating unscoped `finance_categories`, seeded with one row per distinct existing category as `unclassified`, `essential = 0`. `classification` in `needs | wants | savings | income | debt | other | unclassified`. Analytics `LEFT JOIN finance_categories c ON c.name = t.category`; a missing join means unclassified and non-essential.

No foreign key on `transactions.category`: that column is written by the REST API, MCP tools, and the offline replay queue as free text, and a foreign key would break both free-text entry and replay of a category whose row does not exist yet.

Settings surface to classify categories. The frontend's nine hardcoded `PRESET_CATEGORIES` in `apps/web/src/features/finance/schemas/transaction.schemas.ts:9` become seed data, not the source of truth.

## Phase 2 — analytics core

Application services with SQL `GROUP BY`, minor units, explicit `currency`, per-currency grouping: spending summary, cash-flow summary, category trend, budget health, goal health. Savings rate needs a documented definition with a zero-income guard. Every response carries the assumptions it made. Do not claim a trend from a short noisy sample.

## Phase 3 — derived analytics

Monthly digest, recurring-charge summary, spending anomalies (median/MAD, mandatory human-readable `explanation`, no opaque scores), emergency-fund status, affordability check. Affordability returns a financial model with named assumptions, never a yes/no. Recurring-charge summary classifies against explicit bills as `tracked` / `untracked` / `amount_changed`, and never claims a charge is unused without usage data. Bill reconciliation fields land here.

## Phase 4 — MCP surface ✅

Complete on `v1.2.0`:

- `e19c756` split the flat `mcp/server.go` into per-domain files (`tools_finance.go`, `tools_habits.go`, `resources.go`, `prompts.go`, `tools_analytics.go`) — pure move, zero behavior change.
- `f4afc28` fixed `finance_pay_bill`: `MarkPaid` no longer nulls an existing transaction link when called without a transaction ID; added the `create_transaction` flag (default false) that books the expense transaction and payment in one `BeginTx`; corrected the destructive annotation (it is a writable upsert).
- `ad9c5c3` registered the 12 analytics tools (`finance_spending_summary`, `finance_cash_flow_summary`, `finance_category_trend`, `finance_budget_health`, `finance_goal_health`, `finance_savings_rate`, `finance_anomalies`, `finance_recurring_charges`, `finance_bill_reconciliation`, `finance_emergency_fund`, `finance_affordability`, `finance_monthly_digest`) with `readOnlyHint` + `idempotentHint`; rewrote the 3 prompts to reference analytics tools with zero numbers.
- `462de02` added the `finance_classify_category` write tool; updated `docs/mcp.md`.

Surface is 41 tools (22 read + 19 write). User decision: keep all 12 raw list reads, add the analytics tools — do not fold or drop. Read-only mode, loopback defaults, bearer auth, and the name/duration/outcome logging policy preserved. Error-string audit: the only amount-bearing error is `ErrInsufficientClassification` (unclassified percentage + category names, no monetary amounts); it is surfaced as an MCP error.

## Phase 5 — skills, agent profile, dashboard, docs

`skills/` gains finance-core, spending-review, budget-planner, financial-goals, monthly-digest, bill-audit, and affordability, using standard Agent Skills conventions so they are not Hermes-specific. `agent/finance/SOUL.md` plus README, no secrets. The dashboard consumes the same analytics endpoints — no parallel frontend math; `apps/web/src/features/finance/routes/` and `lib/` are currently empty and are the natural home for an overview page. Zod schemas for every new response, no `as` casts. Insufficient-history states say so rather than rendering fabricated zeros. Update `docs/mcp.md` and add `docs/finance-agent.md`.

---

## Validation

Per phase, from repo root:

```bash
mise run test
mise run lint
mise run typecheck
mise run build
mise run build:mcp
mise run migrate   # against a clean database
```

Do not report a phase complete without running these.

## Notes for the implementer

- Read `AGENTS.md` first, then call `mem_context` and `mem_search` for finance decisions before changing a subsystem.
- Relevant repo skills: `go-ddd-api`, `test-quality`, `frontend-zod-contracts`, `tanstack-router-query`, `frontend-shadcn`, `security-review`, `git-commit-and-push`.
- Engram holds the full audit under "Finance agent-native proposal review" and the rulings under "Finance agent-native: decided plan and design rulings".
- Preserve unrelated working-tree changes. Do not push, tag, or release without explicit instruction.
- Keep business logic out of HTTP and MCP handlers; MCP handlers are currently thin and should stay that way.

## Success criteria

- Date-range analytics agree with the user's Manila calendar at day boundaries.
- A retried write creates one row, not two.
- No aggregate silently sums across currencies.
- Budget health and bill audit do not contradict each other; divergence is reported, not hidden.
- Agents answer the common questions without fetching raw transaction dumps.
- Wallet sums are never labeled net worth.
