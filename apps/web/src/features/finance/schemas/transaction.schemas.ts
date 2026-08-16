import { z } from 'zod'

export const TransactionTypeSchema = z.enum(['expense', 'income'])
export type TransactionType = z.infer<typeof TransactionTypeSchema>

export const CategorySchema = z.string().min(1, 'Category is required')
export type Category = z.infer<typeof CategorySchema>

export const CreateTransactionSchema = z.object({
  amountCents: z.number().int().positive('Amount must be positive'),
  category: CategorySchema,
  description: z.string().default(''),
  type: TransactionTypeSchema,
  walletId: z.string(),
  transactionDate: z.string().optional(),
  // Client-generated per logical mutation; the server dedupes replays on it.
  idempotencyKey: z.string().optional(),
})
export type CreateTransaction = z.infer<typeof CreateTransactionSchema>

export const TransactionSchema = z.object({
  id: z.string(),
  amountCents: z.number().int(),
  currency: z.string(),
  category: z.string(),
  description: z.string(),
  type: TransactionTypeSchema,
  walletId: z.string(),
  walletName: z.string().optional(),
  transactionDate: z.string(),
  createdAt: z.string(),
  updatedAt: z.string().optional(),
  // Optimistic concurrency: the server rejects stale edits/deletes unless the
  // client sends this revision in If-Match.
  revision: z.number().int().default(1),
  // True when the transaction was booked from a statement import; the
  // original statement entry stays immutable.
  imported: z.boolean().default(false),
  importProvider: z.string().optional(),
})
export type Transaction = z.infer<typeof TransactionSchema>

// Partial edit payload: every field optional, at least one required. Fields
// are the same shape as CreateTransactionSchema minus idempotencyKey.
export const UpdateTransactionSchema = z
  .object({
    amountCents: z.number().int().positive('Amount must be positive').optional(),
    category: CategorySchema.optional(),
    description: z.string().optional(),
    type: TransactionTypeSchema.optional(),
    walletId: z.string().optional(),
    transactionDate: z.string().optional(),
  })
  .refine((v) => Object.values(v).some((x) => x !== undefined), {
    message: 'At least one field is required',
  })
export type UpdateTransaction = z.infer<typeof UpdateTransactionSchema>

export const TransactionListSchema = z.object({
  data: z.array(TransactionSchema),
})
export type TransactionList = z.infer<typeof TransactionListSchema>

// Bulk ops: one shared patch applied to many transactions, each carrying the
// revision the client last saw (per-row optimistic concurrency).
export const BulkTransactionItemSchema = z.object({
  id: z.string(),
  revision: z.number().int().positive(),
})
export type BulkTransactionItem = z.infer<typeof BulkTransactionItemSchema>

// Only category/type/wallet are bulk-editable: amount, date, and description
// are intentionally excluded because mass-applying them is almost always a
// mistake. At least one field is required.
export const BulkUpdatePatchSchema = z
  .object({
    category: CategorySchema.optional(),
    type: TransactionTypeSchema.optional(),
    walletId: z.string().optional(),
  })
  .refine((v) => Object.values(v).some((x) => x !== undefined), {
    message: 'At least one field is required',
  })
export type BulkUpdatePatch = z.infer<typeof BulkUpdatePatchSchema>

export const BulkUpdateRequestSchema = z.object({
  items: z.array(BulkTransactionItemSchema).min(1),
  patch: BulkUpdatePatchSchema,
})
export type BulkUpdateRequest = z.infer<typeof BulkUpdateRequestSchema>

export const BulkDeleteRequestSchema = z.object({
  items: z.array(BulkTransactionItemSchema).min(1),
})
export type BulkDeleteRequest = z.infer<typeof BulkDeleteRequestSchema>

export const BulkUpdateResponseSchema = z.object({
  data: z.object({
    updated: z.number().int(),
    transactions: z.array(TransactionSchema),
  }),
})
export type BulkUpdateResponse = z.infer<typeof BulkUpdateResponseSchema>

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