---
name: bill-audit
description: Audit bills — reconciliation against actuals, recurring charges, upcoming bills, and payments.
compatibility: opencode
---

# Bill-audit

Use:
- `finance_bill_reconciliation` for expected vs paid per bill for a month
- `finance_recurring_charges` for recurring charges classified against explicit bills
- `finance_upcoming_bills` for upcoming occurrences and payment status
- `finance_pay_bill` to write a paid payment record

Reconciliation:
- report expected, paid, and missing per bill
- a recurring charge without a matching bill is a gap; surface it
- report per currency

Writes:
- `finance_pay_bill` writes a paid record and may book the expense transaction
  atomically via `create_transaction` — confirm the amount, date, and whether
  the transaction should be booked
- never mark a bill paid without confirmation

Rules:
- never delete a bill without explicit confirmation
- never invent a payment date