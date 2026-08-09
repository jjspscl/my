---
name: affordability
description: Evaluate purchase affordability as a runway model and review emergency fund coverage.
compatibility: opencode
---

# Affordability

Use:
- `finance_affordability` for the runway model before and after a prospective purchase
- `finance_emergency_fund` for liquid balance vs an essential-spend target range

Affordability:
- the result is a runway model, never a yes/no
- report runway before and after the purchase
- surface `assumptions` verbatim
- report per currency

Emergency fund:
- compare liquid balance against the target range in months of essential spending
- report months of runway, not a verdict
- report per currency

Rules:
- never state "you can afford it" or "you cannot afford it"
- never compute runway across currencies
- never omit the `targetMonths` context