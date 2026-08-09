---
name: financial-goals
description: Track and manage savings goals — progress, creation, updates, and contributions.
compatibility: opencode
---

# Financial-goals

Use:
- `finance_goal_health` for goal progress snapshots
- `finance_create_goal` to create a savings goal
- `finance_update_goal` to update a savings goal
- `finance_add_goal_contribution` to add a contribution, optionally with a transfer

Progress:
- report target, current, and remaining in minor units
- never compute progress across goals or currencies
- a goal without contributions is at zero, not missing

Writes:
- confirm goal name, target, and currency before creating
- confirm contribution amount and source wallet before adding
- `finance_add_goal_contribution` may create a transfer; confirm the transfer
  side effect explicitly

Rules:
- never delete or archive a goal without explicit confirmation
- never invent contribution history