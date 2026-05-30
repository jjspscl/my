import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import { BudgetSummarySchema, UpsertBudgetSchema, type UpsertBudget } from '../schemas/budget.schemas'

const BudgetSummaryDataSchema = z.object({
  ok: z.boolean().optional(),
  data: BudgetSummarySchema,
})

const UpsertResponseDataSchema = z.object({
  ok: z.boolean().optional(),
  data: z.object({
    id: z.string(),
    month: z.string(),
  }),
})

export async function getBudgetSummary(month: string): Promise<import('../schemas/budget.schemas').BudgetSummary> {
  const res = await apiClient(`/api/v1/finance/budgets?month=${month}`, BudgetSummaryDataSchema)
  return res.data
}

export async function upsertBudget(data: UpsertBudget) {
  const parsed = UpsertBudgetSchema.parse(data)
  const res = await apiClient('/api/v1/finance/budgets', UpsertResponseDataSchema, {
    method: 'PUT',
    body: JSON.stringify(parsed),
  })
  return res.data as { id: string; month: string }
}