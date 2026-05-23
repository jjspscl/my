# Zod Contracts

## Rule

All frontend application data types MUST be inferred from Zod schemas.

## Allowed

```ts
import { z } from 'zod'

export const TransactionSchema = z.object({
  id: z.string().uuid(),
  amount: z.number().int(),
})

export type Transaction = z.infer<typeof TransactionSchema>
```

## Forbidden

```ts
// ❌ Manual interface
interface Transaction { id: string; amount: number }

// ❌ Manual type alias
type Transaction = { id: string; amount: number }

// ❌ Unsafe cast
const data = await response.json() as Transaction[]
```

## Required Schemas For

- API request/response DTOs
- TanStack Query data
- TanStack Mutation variables
- Route search params
- Form values
- Widget configuration
- Offline sync payloads
- LocalStorage/IndexedDB reads
- Server error envelopes

## Transforms

When schemas use transforms/coercion/defaults:

```ts
export type SearchInput = z.input<typeof SearchSchema>
export type Search = z.output<typeof SearchSchema>
```