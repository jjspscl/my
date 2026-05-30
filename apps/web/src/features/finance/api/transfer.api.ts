import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import { WalletTransferSchema, CreateTransferSchema, type CreateTransfer } from '../schemas/transfer.schemas'

const TransferListDataSchema = z.object({
  data: z.array(WalletTransferSchema),
})

const TransferDataSchema = z.object({
  ok: z.boolean().optional(),
  data: WalletTransferSchema,
})

export async function listTransfers(): Promise<import('../schemas/transfer.schemas').WalletTransfer[]> {
  const res = await apiClient('/api/v1/finance/transfers', TransferListDataSchema)
  return res.data
}

export async function createTransfer(data: CreateTransfer): Promise<import('../schemas/transfer.schemas').WalletTransfer> {
  const parsed = CreateTransferSchema.parse(data)
  const res = await apiClient('/api/v1/finance/transfers', TransferDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data as import('../schemas/transfer.schemas').WalletTransfer
}
