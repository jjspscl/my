import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import { WalletSchema, CreateWalletSchema, type CreateWallet } from '../schemas/wallet.schemas'

const WalletListDataSchema = z.object({
  data: z.array(WalletSchema),
})

const WalletDataSchema = z.object({
  ok: z.boolean().optional(),
  data: WalletSchema,
})

const DeleteResponseSchema = z.object({
  ok: z.boolean().optional(),
})

export async function listWallets(): Promise<import('../schemas/wallet.schemas').Wallet[]> {
  const res = await apiClient('/api/v1/finance/wallets', WalletListDataSchema)
  return res.data
}

export async function createWallet(data: CreateWallet): Promise<import('../schemas/wallet.schemas').Wallet> {
  const parsed = CreateWalletSchema.parse(data)
  const res = await apiClient('/api/v1/finance/wallets', WalletDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
  return res.data as import('../schemas/wallet.schemas').Wallet
}

export async function updateWallet(id: string, data: CreateWallet): Promise<import('../schemas/wallet.schemas').Wallet> {
  const parsed = CreateWalletSchema.parse(data)
  const res = await apiClient(`/api/v1/finance/wallets/${id}`, WalletDataSchema, {
    method: 'PUT',
    body: JSON.stringify(parsed),
  })
  return res.data as import('../schemas/wallet.schemas').Wallet
}

export async function archiveWallet(id: string): Promise<void> {
  await apiClient(`/api/v1/finance/wallets/${id}`, DeleteResponseSchema, {
    method: 'DELETE',
  })
}
