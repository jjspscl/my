import { apiClient } from '@/shared/api/client'
import { randomUUID } from '@/shared/lib/uuid'
import { z } from 'zod'
import { GoalSummarySchema, CreateGoalSchema, type CreateGoal } from '../schemas/goal.schemas'
import { financeMutate, type MutateResult } from './mutate'

const GoalSummaryListDataSchema = z.object({
  data: z.array(GoalSummarySchema),
})

const GoalDataSchema = z.object({
  ok: z.boolean().optional(),
  data: z.object({
    id: z.string(),
    name: z.string(),
    targetAmountCents: z.number().int(),
    targetDate: z.string().nullable().optional(),
    createdAt: z.string(),
    updatedAt: z.string(),
  }),
})

const DeleteResponseSchema = z.object({
  ok: z.boolean().optional(),
})

const ContributionResponseSchema = z.object({
  ok: z.boolean().optional(),
  data: z.object({
    id: z.string(),
    goalId: z.string(),
    amountCents: z.number().int(),
    contributedAt: z.string(),
    note: z.string().nullable(),
  }),
})

export async function listGoals(): Promise<import('../schemas/goal.schemas').GoalSummary[]> {
  const res = await apiClient('/api/v1/finance/goals', GoalSummaryListDataSchema)
  return res.data
}

export async function createGoal(data: CreateGoal): Promise<import('../schemas/goal.schemas').SavingsGoal> {
  const parsed = CreateGoalSchema.parse(data)
  const res = await apiClient('/api/v1/finance/goals', GoalDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data as import('../schemas/goal.schemas').SavingsGoal
}

export async function updateGoal(id: string, data: CreateGoal): Promise<import('../schemas/goal.schemas').SavingsGoal> {
  const parsed = CreateGoalSchema.parse(data)
  const res = await apiClient(`/api/v1/finance/goals/${id}`, GoalDataSchema, {
    method: 'PUT',
    body: JSON.stringify(parsed),
  })
  return res.data as import('../schemas/goal.schemas').SavingsGoal
}

export async function deleteGoal(id: string): Promise<void> {
  await apiClient(`/api/v1/finance/goals/${id}`, DeleteResponseSchema, {
    method: 'DELETE',
  })
}

export async function addContribution(goalId: string, amountCents: number, contributedAt: string, note = '', sourceWalletId?: string, idempotencyKey?: string): Promise<MutateResult<unknown>> {
  return financeMutate(`/api/v1/finance/goals/${goalId}/contributions`, {
    amountCents,
    contributedAt,
    note,
    sourceWalletId,
    idempotencyKey: idempotencyKey ?? randomUUID(),
  }, ContributionResponseSchema)
}