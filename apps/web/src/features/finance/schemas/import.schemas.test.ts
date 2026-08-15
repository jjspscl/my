import { describe, expect, it } from 'vitest'

import { CreateImportRowSchema, CreateImportSchema } from './import.schemas'

function row(overrides: Record<string, unknown> = {}) {
  return {
    sourceReference: 'REF1',
    occurredAt: '2026-07-01T09:30:00+08:00',
    amountCents: 1500,
    kind: 'expense',
    category: 'Food',
    description: 'Jollibee',
    ...overrides,
  }
}

describe('CreateImportRowSchema counter wallet rules', () => {
  it('accepts a plain expense without a counter wallet', () => {
    expect(CreateImportRowSchema.safeParse(row()).success).toBe(true)
  })

  it('rejects a transfer without a counter wallet', () => {
    const res = CreateImportRowSchema.safeParse(row({ kind: 'transfer_in' }))
    expect(res.success).toBe(false)
  })

  it('accepts a transfer with a counter wallet', () => {
    expect(CreateImportRowSchema.safeParse(row({ kind: 'transfer_in', counterWalletId: 'w2' })).success).toBe(true)
  })

  it('rejects expense/income carrying a counter wallet', () => {
    expect(CreateImportRowSchema.safeParse(row({ counterWalletId: 'w2' })).success).toBe(false)
    expect(CreateImportRowSchema.safeParse(row({ kind: 'income', counterWalletId: 'w2' })).success).toBe(false)
  })
})

describe('CreateImportSchema', () => {
  it('requires wallet or createWallet selection', () => {
    const base = {
      provider: 'gcash_pdf' as const,
      fileFingerprint: 'a'.repeat(64),
      statementFrom: '2026-07-01',
      statementTo: '2026-07-31',
      openingBalanceCents: 0,
      endingBalanceCents: 1000,
      reconciliation: 'ok' as const,
      rows: [row()],
    }
    expect(CreateImportSchema.safeParse(base).success).toBe(false)
    expect(CreateImportSchema.safeParse({ ...base, walletId: 'w1' }).success).toBe(true)
    expect(
      CreateImportSchema.safeParse({ ...base, createWallet: { name: 'GCash', openingBalanceCents: 0 } }).success,
    ).toBe(true)
  })
})
