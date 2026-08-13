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

  it('analytics returns key with analytics suffix', () => {
    expect(financeKeys.analytics()).toEqual(['finance', 'analytics'])
  })

  it('analyticsQuery returns key with query name', () => {
    expect(financeKeys.analyticsQuery('spending-summary')).toEqual([
      'finance',
      'analytics',
      'spending-summary',
      undefined,
    ])
  })

  it('analyticsQuery with filters returns key with filters object', () => {
    const filters = { from: '2026-01-01', to: '2026-05-23' }
    expect(financeKeys.analyticsQuery('spending-summary', filters)).toEqual([
      'finance',
      'analytics',
      'spending-summary',
      filters,
    ])
  })

  it('analyticsQuery with different filters produce different keys', () => {
    const f1 = financeKeys.analyticsQuery('spending-summary', { from: '2026-01-01' })
    const f2 = financeKeys.analyticsQuery('spending-summary', { from: '2026-02-01' })
    expect(f1).not.toEqual(f2)
  })

  it('analyticsQuery with different names produce different keys', () => {
    const f1 = financeKeys.analyticsQuery('spending-summary')
    const f2 = financeKeys.analyticsQuery('cash-flow-summary')
    expect(f1).not.toEqual(f2)
  })

  it('analytics and transactions produce different keys', () => {
    expect(financeKeys.analytics()).not.toEqual(financeKeys.transactions())
  })
})