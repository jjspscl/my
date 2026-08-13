---
name: finance-core
description: Shared conventions for all finance agent skills — units, dates, currencies, insufficient data, and safety rules.
compatibility: opencode
---

# finance-core

Load before any other finance skill.

Use:
- `finance_today_total` for today's income and expense totals
- `finance_list_wallets` for wallet balances

Amounts:
- always minor units (cents), never currency decimals
- never sum across currencies; report per currency
- wallet balances are liquid balance, not net worth
- no debt, credit, or investment netting

Dates:
- months as `YYYY-MM`
- timestamps as RFC3339

Classification:
- `insufficient_classification` means too little history to classify
- `sufficient: false` results are not definitive; say so
- surface `assumptions` and `explanation` fields verbatim

Never:
- present wallet sums as net worth
- invent amounts, categories, or history
- write without explicit user confirmation
- store or repeat secrets, tokens, or personal data