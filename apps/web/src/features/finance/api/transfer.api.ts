import { apiClient } from '@/shared/api/client'
import { randomUUID } from '@/shared/lib/uuid'
import { z } from 'zod'
import { WalletTransferSchema, CreateTransferSchema, type CreateTransfer } from '../schemas/transfer.schemas'
import { financeMutate, isQueued, type MutateResult } from './mutate'

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

export async function createTransfer(data: CreateTransfer): Promise<MutateResult<import('../schemas/transfer.schemas').WalletTransfer>> {
  const parsed = CreateTransferSchema.parse({
    ...data,
    idempotencyKey: data.idempotencyKey ?? randomUUID(),
  })
  const res = await financeMutate('/api/v1/finance/transfers', parsed, TransferDataSchema)
  if (isQueued(res)) return res
  return res.data as import('../schemas/transfer.schemas').WalletTransfer
}
