import { apiClient } from '@/shared/api/client'
import { randomUUID } from '@/shared/lib/uuid'
import {
  ApiOKResponseSchema,
  type BulkDeleteRequest,
  BulkDeleteRequestSchema,
  type BulkUpdateRequest,
  BulkUpdateRequestSchema,
  type BulkUpdateResponse,
  BulkUpdateResponseSchema,
  type CreateTransaction,
  CreateTransactionSchema,
  DailyTotalSchema,
  TransactionSchema,
  type UpdateTransaction,
  UpdateTransactionSchema,
} from '../schemas/transaction.schemas'
import { financeMutate, type MutateResult } from './mutate'
import { z } from 'zod'

const TransactionListDataSchema = z.object({ data: z.array(TransactionSchema) })
const DailyTotalDataSchema = z.object({ data: DailyTotalSchema })
const TransactionDataSchema = z.object({ data: TransactionSchema })

export function createTransaction(data: CreateTransaction): Promise<MutateResult<unknown>> {
  const parsed = CreateTransactionSchema.parse({
    ...data,
    idempotencyKey: data.idempotencyKey ?? randomUUID(),
  })
  return financeMutate('/api/v1/finance/transactions', parsed, z.any())
}

// updateTransaction patches a transaction with an If-Match precondition on the
// revision the client last saw; the server answers 412 when stale. Online-only:
// the offline queue has no conflict protocol, so edits are disabled offline.
export function updateTransaction(
  id: string,
  data: UpdateTransaction,
  revision: number,
): Promise<z.infer<typeof TransactionSchema>> {
  const parsed = UpdateTransactionSchema.parse(data)
  return apiClient(`/api/v1/finance/transactions/${id}`, TransactionDataSchema, {
    method: 'PATCH',
    headers: { 'If-Match': `"${revision}"` },
    body: JSON.stringify(parsed),
  }).then((res) => res.data)
}

export async function listTransactions(from?: string, to?: string) {
  const params = new URLSearchParams()
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  const qs = params.toString()
  const res = await apiClient(
    `/api/v1/finance/transactions${qs ? `?${qs}` : ''}`,
    TransactionListDataSchema,
  )
  return res.data
}

export async function getTodayTotal() {
  const res = await apiClient('/api/v1/finance/transactions/today-total', DailyTotalDataSchema)
  return res.data
}

// deleteTransaction removes a transaction. When revision is provided the
// server enforces If-Match and answers 412 when the row moved on.
export function deleteTransaction(id: string, revision?: number) {
  return apiClient(`/api/v1/finance/transactions/${id}`, ApiOKResponseSchema, {
    method: 'DELETE',
    ...(revision ? { headers: { 'If-Match': `"${revision}"` } } : {}),
  })
}

// bulkUpdateTransactions applies one patch to many transactions atomically;
// the server answers 412 when any item's revision is stale and nothing is
// applied. Online-only (no conflict protocol in the offline queue).
export async function bulkUpdateTransactions(
  data: BulkUpdateRequest,
): Promise<BulkUpdateResponse['data']> {
  const parsed = BulkUpdateRequestSchema.parse(data)
  const res = await apiClient('/api/v1/finance/transactions/bulk', BulkUpdateResponseSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data
}

// bulkDeleteTransactions removes many transactions atomically.
export async function bulkDeleteTransactions(data: BulkDeleteRequest): Promise<number> {
  const parsed = BulkDeleteRequestSchema.parse(data)
  const res = await apiClient(
    '/api/v1/finance/transactions/bulk-delete',
    z.object({ data: z.object({ deleted: z.number().int() }) }),
    { method: 'POST', body: JSON.stringify(parsed) },
  )
  return res.data.deleted
}