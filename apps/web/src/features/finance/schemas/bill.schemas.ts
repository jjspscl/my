import { z } from 'zod'

export const FrequencySchema = z.enum(['monthly', 'weekly', 'yearly'])
export type Frequency = z.infer<typeof FrequencySchema>

export const OccurrenceStatusSchema = z.enum(['pending', 'overdue', 'paid', 'skipped'])
export type OccurrenceStatus = z.infer<typeof OccurrenceStatusSchema>

export const RecurringBillSchema = z.object({
  id: z.string(),
  name: z.string().min(1, 'Name is required'),
  category: z.string().min(1, 'Category is required'),
  amountCents: z.number().int().positive('Amount must be positive'),
  frequency: FrequencySchema,
  dayOfMonth: z.number().int().min(1).max(31),
  startDate: z.string(),
  endDate: z.string().nullable().optional(),
  autoMatch: z.boolean().default(false),
  matchPattern: z.string().nullable().optional(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type RecurringBill = z.infer<typeof RecurringBillSchema>

export const CreateBillSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  category: z.string().min(1, 'Category is required'),
  amountCents: z.number().int().positive('Amount must be positive'),
  frequency: FrequencySchema,
  dayOfMonth: z.number().int().min(1, 'Day is required').max(31),
  startDate: z.string(),
  endDate: z.string().nullable().optional(),
  autoMatch: z.boolean().default(false),
  matchPattern: z.string().nullable().optional(),
})
export type CreateBill = z.infer<typeof CreateBillSchema>

export const UpcomingBillSchema = z.object({
  id: z.string(),
  name: z.string(),
  category: z.string(),
  amountCents: z.number().int(),
  frequency: FrequencySchema,
  dayOfMonth: z.number().int(),
  dueDate: z.string(),
  status: OccurrenceStatusSchema,
  paidAmountCents: z.number().int().nullable().optional(),
  paidDate: z.string().nullable().optional(),
  autoMatch: z.boolean(),
})
export type UpcomingBill = z.infer<typeof UpcomingBillSchema>