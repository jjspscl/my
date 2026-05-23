import { apiClient } from '@/shared/api/client'
import {
  ApiOKResponseSchema,
  type CreateTransaction,
  CreateTransactionSchema,
  DailyTotalSchema,
  TransactionSchema,
} from '../schemas/transaction.schemas'
import { z } from 'zod'

const TransactionListDataSchema = z.object({ data: z.array(TransactionSchema) })
const DailyTotalDataSchema = z.object({ data: DailyTotalSchema })

export function createTransaction(data: CreateTransaction) {
  const parsed = CreateTransactionSchema.parse(data)
  return apiClient('/api/v1/finance/transactions', z.any(), {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
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

export function deleteTransaction(id: string) {
  return apiClient(`/api/v1/finance/transactions/${id}`, ApiOKResponseSchema, {
    method: 'DELETE',
  })
}