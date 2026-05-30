import { z } from 'zod'

export const WalletKindSchema = z.enum(['cash', 'bank', 'ewallet'])
export type WalletKind = z.infer<typeof WalletKindSchema>

export const WalletSchema = z.object({
  id: z.string(),
  name: z.string().min(1, 'Name is required'),
  kind: WalletKindSchema,
  currency: z.string().default('PHP'),
  openingBalanceCents: z.number().int().min(0),
  balanceCents: z.number().int(),
  incomeCents: z.number().int(),
  expenseCents: z.number().int(),
  incomingTransferCents: z.number().int(),
  outgoingTransferCents: z.number().int(),
  isDefault: z.boolean().default(false),
  archivedAt: z.string().nullable().optional(),
  createdAt: z.string(),
  updatedAt: z.string(),
})
export type Wallet = z.infer<typeof WalletSchema>

export const CreateWalletSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  kind: WalletKindSchema,
  openingBalanceCents: z.number().int().min(0).default(0),
})
export type CreateWallet = z.infer<typeof CreateWalletSchema>

export const UpdateWalletSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  kind: WalletKindSchema,
  openingBalanceCents: z.number().int().min(0).default(0),
})
export type UpdateWallet = z.infer<typeof UpdateWalletSchema>
