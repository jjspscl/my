import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import { RecurringBillSchema, CreateBillSchema, UpcomingBillSchema, type CreateBill } from '../schemas/bill.schemas'

const RecurringBillDataSchema = z.object({
  ok: z.boolean().optional(),
  data: RecurringBillSchema,
})

const RecurringBillListDataSchema = z.object({
  data: z.array(RecurringBillSchema),
})

const UpcomingBillListDataSchema = z.object({
  data: z.array(UpcomingBillSchema),
})

const DeleteResponseSchema = z.object({
  ok: z.boolean().optional(),
})

const MarkPaidResponseSchema = z.object({
  ok: z.boolean().optional(),
  data: z.object({
    id: z.string(),
    billId: z.string(),
    dueDate: z.string(),
    status: z.string(),
    paidDate: z.string().nullable(),
  }),
})

export async function listBills(): Promise<import('../schemas/bill.schemas').RecurringBill[]> {
  const res = await apiClient('/api/v1/finance/bills', RecurringBillListDataSchema)
  return res.data
}

export async function createBill(data: CreateBill): Promise<import('../schemas/bill.schemas').RecurringBill> {
  const parsed = CreateBillSchema.parse(data)
  const res = await apiClient('/api/v1/finance/bills', RecurringBillDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function updateBill(id: string, data: CreateBill): Promise<import('../schemas/bill.schemas').RecurringBill> {
  const parsed = CreateBillSchema.parse(data)
  const res = await apiClient(`/api/v1/finance/bills/${id}`, RecurringBillDataSchema, {
    method: 'PUT',
    body: JSON.stringify(parsed),
  })
  return res.data
}

export async function deleteBill(id: string): Promise<void> {
  await apiClient(`/api/v1/finance/bills/${id}`, DeleteResponseSchema, {
    method: 'DELETE',
  })
}

export async function getUpcomingBills(days = 30): Promise<import('../schemas/bill.schemas').UpcomingBill[]> {
  const res = await apiClient(`/api/v1/finance/bills/upcoming?days=${days}`, UpcomingBillListDataSchema)
  return res.data
}

export async function markBillPaid(billId: string, dueDate: string, transactionId?: string): Promise<void> {
  await apiClient(`/api/v1/finance/bills/${billId}/pay`, MarkPaidResponseSchema, {
    method: 'POST',
    body: JSON.stringify({ dueDate, transactionId }),
  })
}