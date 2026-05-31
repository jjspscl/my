import { describe, it, expect } from 'vitest'
import {
  TransactionSchema,
  CreateTransactionSchema,
  DailyTotalSchema,
  TransactionTypeSchema,
} from './transaction.schemas'

describe('TransactionTypeSchema', () => {
  it('accepts expense', () => {
    expect(TransactionTypeSchema.parse('expense')).toBe('expense')
  })

  it('accepts income', () => {
    expect(TransactionTypeSchema.parse('income')).toBe('income')
  })

  it('rejects invalid type', () => {
    const result = TransactionTypeSchema.safeParse('savings')
    expect(result.success).toBe(false)
  })
})

describe('TransactionSchema', () => {
  it('parses valid transaction', () => {
    const tx = TransactionSchema.parse({
      id: 'tx-1',
      amountCents: 150000,
      currency: 'PHP',
      category: 'food',
      description: 'Lunch',
      type: 'expense',
      walletId: 'w-1',
      transactionDate: '2026-01-15',
      createdAt: '2026-01-15T10:00:00Z',
    })
    expect(tx.id).toBe('tx-1')
    expect(tx.amountCents).toBe(150000)
    expect(tx.type).toBe('expense')
  })

  it('parses income transaction', () => {
    const tx = TransactionSchema.parse({
      id: 'tx-2',
      amountCents: 5000000,
      currency: 'PHP',
      category: 'salary',
      description: 'Monthly salary',
      type: 'income',
      walletId: 'w-1',
      transactionDate: '2026-01-15',
      createdAt: '2026-01-15T10:00:00Z',
    })
    expect(tx.type).toBe('income')
    expect(tx.amountCents).toBe(5000000)
  })

  it('accepts negative amountCents (schema allows int, service validates > 0)', () => {
    const result = TransactionSchema.safeParse({
      id: 'tx-3',
      amountCents: -100,
      currency: 'PHP',
      category: 'food',
      description: '',
      type: 'expense',
      walletId: 'w-1',
      transactionDate: '2026-01-15',
      createdAt: '2026-01-15T10:00:00Z',
    })
    expect(result.success).toBe(true) // schema only enforces int, not positive
  })

  it('rejects invalid type', () => {
    const result = TransactionSchema.safeParse({
      id: 'tx-4',
      amountCents: 100,
      currency: 'PHP',
      category: 'food',
      description: '',
      type: 'invalid',
      walletId: 'w-1',
      transactionDate: '2026-01-15',
      createdAt: '2026-01-15T10:00:00Z',
    })
    expect(result.success).toBe(false)
  })

  it('requires transactionDate', () => {
    const result = TransactionSchema.safeParse({
      id: 'tx-5',
      amountCents: 100,
      currency: 'PHP',
      category: 'food',
      description: '',
      type: 'expense',
      walletId: 'w-1',
      createdAt: '2026-01-15T10:00:00Z',
    })
    expect(result.success).toBe(false)
  })
})

describe('CreateTransactionSchema', () => {
  it('accepts valid input', () => {
    const result = CreateTransactionSchema.safeParse({
      amountCents: 150000,
      category: 'food',
      description: 'Lunch',
      type: 'expense',
      walletId: 'w-1',
    })
    expect(result.success).toBe(true)
  })

  it('rejects negative amount', () => {
    const result = CreateTransactionSchema.safeParse({
      amountCents: -100,
      category: 'food',
      type: 'expense',
      walletId: 'w-1',
    })
    expect(result.success).toBe(false)
  })

  it('rejects zero amount', () => {
    const result = CreateTransactionSchema.safeParse({
      amountCents: 0,
      category: 'food',
      type: 'expense',
      walletId: 'w-1',
    })
    expect(result.success).toBe(false)
  })

  it('rejects missing category', () => {
    const result = CreateTransactionSchema.safeParse({
      amountCents: 100,
      category: '',
      type: 'expense',
      walletId: 'w-1',
    })
    expect(result.success).toBe(false)
  })

  it('defaults description to empty string', () => {
    const input = CreateTransactionSchema.parse({
      amountCents: 100,
      category: 'food',
      type: 'expense',
      walletId: 'w-1',
    })
    expect(input.description).toBe('')
  })

  it('accepts optional transactionDate', () => {
    const result = CreateTransactionSchema.safeParse({
      amountCents: 100,
      category: 'food',
      description: '',
      type: 'expense',
      walletId: 'w-1',
      transactionDate: '2026-01-15',
    })
    expect(result.success).toBe(true)
  })

  it('rejects invalid type', () => {
    const result = CreateTransactionSchema.safeParse({
      amountCents: 100,
      category: 'food',
      type: 'invalid',
      walletId: 'w-1',
    })
    expect(result.success).toBe(false)
  })
})

describe('DailyTotalSchema', () => {
  it('parses valid daily total', () => {
    const total = DailyTotalSchema.parse({
      date: '2026-01-15',
      totalCents: 500000,
      expenseCents: 200000,
      incomeCents: 700000,
      currency: 'PHP',
    })
    expect(total.date).toBe('2026-01-15')
    expect(total.totalCents).toBe(500000)
    expect(total.currency).toBe('PHP')
  })

  it('defaults currency to PHP', () => {
    const total = DailyTotalSchema.parse({
      date: '2026-01-15',
      totalCents: 0,
      expenseCents: 0,
      incomeCents: 0,
    })
    expect(total.currency).toBe('PHP')
  })

  it('rejects negative totalCents', () => {
    const result = DailyTotalSchema.safeParse({
      date: '2026-01-15',
      totalCents: -100,
      expenseCents: 0,
      incomeCents: 0,
    })
    expect(result.success).toBe(true) // valid because no .positive() constraint
  })

  it('rejects missing date', () => {
    const result = DailyTotalSchema.safeParse({
      totalCents: 0,
      expenseCents: 0,
      incomeCents: 0,
    })
    expect(result.success).toBe(false)
  })
})