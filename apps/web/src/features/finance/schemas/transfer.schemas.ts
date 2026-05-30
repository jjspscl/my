import { z } from 'zod'

export const WalletTransferSchema = z.object({
  id: z.string(),
  fromWalletId: z.string(),
  toWalletId: z.string(),
  amountCents: z.number().int().positive(),
  description: z.string().default(''),
  transferDate: z.string(),
  createdAt: z.string(),
})
export type WalletTransfer = z.infer<typeof WalletTransferSchema>

export const CreateTransferSchema = z.object({
  fromWalletId: z.string().min(1, 'Source wallet is required'),
  toWalletId: z.string().min(1, 'Destination wallet is required'),
  amountCents: z.number().int().positive('Amount must be positive'),
  description: z.string().default(''),
  transferDate: z.string().min(1, 'Date is required'),
})
export type CreateTransfer = z.infer<typeof CreateTransferSchema>
