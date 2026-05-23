import { z } from 'zod'

export const TransactionTypeSchema = z.enum(['expense', 'income'])
export type TransactionType = z.infer<typeof TransactionTypeSchema>

export const CategorySchema = z.string().min(1, 'Category is required')
export type Category = z.infer<typeof CategorySchema>

export const PRESET_CATEGORIES = [
  'Food',
  'Transport',
  'Groceries',
  'Bills',
  'Entertainment',
  'Health',
  'Shopping',
  'Education',
  'Other',
] as const

export const CreateTransactionSchema = z.object({
  amountCents: z.number().int().positive('Amount must be positive'),
  category: CategorySchema,
  description: z.string().default(''),
  type: TransactionTypeSchema,
  transactionDate: z.string().optional(),
})
export type CreateTransaction = z.infer<typeof CreateTransactionSchema>

export const TransactionSchema = z.object({
  id: z.string(),
  amountCents: z.number().int(),
  currency: z.string(),
  category: z.string(),
  description: z.string(),
  type: TransactionTypeSchema,
  transactionDate: z.string(),
  createdAt: z.string(),
})
export type Transaction = z.infer<typeof TransactionSchema>

export const TransactionListSchema = z.object({
  data: z.array(TransactionSchema),
})
export type TransactionList = z.infer<typeof TransactionListSchema>

export const DailyTotalSchema = z.object({
  date: z.string(),
  totalCents: z.number().int(),
  expenseCents: z.number().int(),
  incomeCents: z.number().int(),
  currency: z.string().default('PHP'),
})
export type DailyTotal = z.infer<typeof DailyTotalSchema>

export const ApiDataResponseSchema = <T extends z.ZodTypeAny>(dataSchema: T) =>
  z.object({
    ok: z.boolean().optional(),
    error: z.string().optional(),
    data: dataSchema.optional(),
  })

export const ApiOKResponseSchema = z.object({
  ok: z.boolean().optional(),
  error: z.string().optional(),
})