import { z } from 'zod'

export const BudgetCategoryInputSchema = z.object({
  category: z.string().min(1, 'Category is required'),
  allocatedCents: z.number().int().min(0, 'Amount cannot be negative'),
  rolloverEnabled: z.boolean().default(false),
})
export type BudgetCategoryInput = z.infer<typeof BudgetCategoryInputSchema>

export const UpsertBudgetSchema = z.object({
  month: z.string().regex(/^\d{4}-(0[1-9]|1[0-2])$/, 'Invalid month format'),
  categories: z.array(BudgetCategoryInputSchema),
})
export type UpsertBudget = z.infer<typeof UpsertBudgetSchema>

export const BudgetCategorySummarySchema = z.object({
  category: z.string(),
  allocatedCents: z.number().int(),
  spentCents: z.number().int(),
  remainingCents: z.number().int(),
  rolloverEnabled: z.boolean(),
})
export type BudgetCategorySummary = z.infer<typeof BudgetCategorySummarySchema>

export const BudgetSummarySchema = z.object({
  month: z.string(),
  totalAllocatedCents: z.number().int(),
  totalSpentCents: z.number().int(),
  totalRemainingCents: z.number().int(),
  categories: z.array(BudgetCategorySummarySchema),
})
export type BudgetSummary = z.infer<typeof BudgetSummarySchema>