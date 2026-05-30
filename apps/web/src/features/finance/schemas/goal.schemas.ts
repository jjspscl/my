import { z } from 'zod'

export const GoalStatusSchema = z.enum(['not_started', 'in_progress', 'achieved', 'behind'])
export type GoalStatus = z.infer<typeof GoalStatusSchema>

export const SavingsGoalSchema = z.object({
  id: z.string(),
  name: z.string().min(1, 'Name is required'),
  targetAmountCents: z.number().int().positive('Target must be positive'),
  targetDate: z.string().nullable().optional(),
  targetWalletId: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type SavingsGoal = z.infer<typeof SavingsGoalSchema>

export const GoalSummarySchema = z.object({
  id: z.string(),
  name: z.string(),
  targetAmountCents: z.number().int(),
  targetDate: z.string().nullable().optional(),
  targetWalletId: z.string(),
  currentAmountCents: z.number().int(),
  remainingAmountCents: z.number().int(),
  progressPercent: z.number().int().min(0).max(100),
  requiredMonthlyCents: z.number().int().nullable().optional(),
  status: GoalStatusSchema,
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type GoalSummary = z.infer<typeof GoalSummarySchema>

export const CreateGoalSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  targetAmountCents: z.number().int().positive('Target must be positive'),
  targetDate: z.string().nullable().optional(),
  targetWalletId: z.string().min(1, 'Target wallet is required'),
})
export type CreateGoal = z.infer<typeof CreateGoalSchema>

export const AddContributionSchema = z.object({
  amountCents: z.number().int().positive('Amount must be positive'),
  contributedAt: z.string().min(1, 'Date is required'),
  note: z.string().optional(),
  sourceWalletId: z.string().optional(),
})
export type AddContribution = z.infer<typeof AddContributionSchema>