---
name: frontend-zod-contracts
description: Enforce schema-first frontend data contracts. All frontend application data types must be inferred from Zod schemas.
compatibility: opencode
---

# frontend-zod-contracts

All frontend application data must come from Zod schemas.

Required for:
- API request DTOs
- API response DTOs
- forms
- route search params
- mutation variables
- query responses
- widget configs
- module manifests
- offline records
- persisted local storage
- sync payloads

Allowed:

export const TransactionSchema = z.object({
  id: z.string().uuid(),
  amountMinor: z.number().int(),
})

export type Transaction = z.infer<typeof TransactionSchema>

Forbidden:

interface Transaction {
  id: string
  amountMinor: number
}

Forbidden:

type Transaction = {
  id: string
  amountMinor: number
}

Forbidden:

const data = await response.json() as Transaction[]

Every API response must parse through Zod before reaching components.
Every localStorage/IndexedDB read must validate with Zod.
