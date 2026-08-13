---
name: budget-planner
description: Plan and review budgets — budget health, budget summary, budget updates, and category classification.
compatibility: opencode
---

# Budget-planner

Use:
- `finance_budget_health` for plan vs actuals, including unbudgeted spending
- `finance_budget_summary` for the budget category summary
- `finance_upsert_budget` to replace a month's budget categories
- `finance_classify_category` to set category classification and flags

Budget health:
- unbudgeted spending is part of the picture; surface it
- report per currency
- a missing budget is not a zero budget; say so

Writes:
- `finance_upsert_budget` replaces the month's categories — confirm the full
  replacement before calling
- `finance_classify_category` sets classification and essential/active flags —
  confirm the flags before calling
- never create a budget without a month

Rules:
- never invent category names
- never reclassify a category without user confirmation