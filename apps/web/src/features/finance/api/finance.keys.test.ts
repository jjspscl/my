import { describe, it, expect } from 'vitest'
import { financeKeys } from './finance.keys'

describe('financeKeys', () => {
  it('all returns base key', () => {
    expect(financeKeys.all).toEqual(['finance'])
  })

  it('transactions returns key with transactions suffix', () => {
    expect(financeKeys.transactions()).toEqual(['finance', 'transactions'])
  })

  it('transactionList returns key with filters', () => {
    expect(financeKeys.transactionList()).toEqual(['finance', 'transactions', 'list', undefined])
  })

  it('transactionList with filters returns key with filters object', () => {
    const filters = { from: '2026-01-01', to: '2026-05-23' }
    expect(financeKeys.transactionList(filters)).toEqual(['finance', 'transactions', 'list', filters])
  })

  it('transactionList with different filters produce different keys', () => {
    const f1 = financeKeys.transactionList({ from: '2026-01-01' })
    const f2 = financeKeys.transactionList({ from: '2026-02-01' })
    expect(f1).not.toEqual(f2)
  })

  it('todayTotal returns key with today-total suffix', () => {
    expect(financeKeys.todayTotal()).toEqual(['finance', 'transactions', 'today-total'])
  })

  it('transactionList and todayTotal produce different keys', () => {
    expect(financeKeys.transactionList()).not.toEqual(financeKeys.todayTotal())
  })
})