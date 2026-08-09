---
name: monthly-digest
description: Produce a composed monthly finance summary from the monthly digest tool.
compatibility: opencode
---

# Monthly-digest

Use:
- `finance_monthly_digest` for the composed monthly summary

Behavior:
- the digest is composed server-side; do not recompute from raw data
- honor `omitted` sections — do not fill them from other tools
- surface `assumptions` and `explanation` fields verbatim
- report per currency

Rules:
- never present a digest section that the tool omitted
- never add commentary beyond what the digest supports
- never fabricate month-over-month change the digest does not contain