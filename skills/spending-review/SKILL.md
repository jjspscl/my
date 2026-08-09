---
name: spending-review
description: Analyze spending, cash flow, savings rate, and anomalies per currency using the finance analytics tools.
compatibility: opencode
---

# Spending-review

Use:
- `finance_spending_summary` for per-currency expense by classification
- `finance_cash_flow_summary` for income/expense/net with monthly series
- `finance_savings_rate` for per-currency savings rate
- `finance_category_trend` for one category's monthly spending series
- `finance_anomalies` for unusual monthly spending per category

Analysis:
- report per currency, never merged
- anomalies are statistical flags, not judgments; explain the flag, not the motive
- savings rate is per currency; do not average across currencies
- compare against prior months from the series, not invented baselines

Never:
- call `finance_category_trend` without a category
- present a single month as a trend
- fabricate a reason for an anomaly