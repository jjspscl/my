import { describe, expect, it } from 'vitest'

import { classifyRow } from './gcash-classifier'
import type { GcashRow } from './gcash-text'

function row(description: string, debit: string, credit: string): GcashRow {
  return {
    dateTime: '2026-07-01 10:00 AM',
    description,
    referenceNo: '1234567890123',
    debit,
    credit,
    balance: '5,000.00',
  }
}

describe('classifyRow', () => {
  it('classifies payments as expenses', () => {
    const c = classifyRow(row('Payment to SM Supermarket', '150.00', ''))
    expect(c.kind).toBe('expense')
    expect(c.category).toBe('Shopping')
  })

  it('classifies received funds as income', () => {
    const c = classifyRow(row('Received from Juan Dela Cruz', '', '2,500.00'))
    expect(c.kind).toBe('income')
  })

  it('detects bank transfer out with wallet hint', () => {
    const c = classifyRow(row('BDO Bank Transfer', '10,000.00', ''))
    expect(c.kind).toBe('transfer_out')
    expect(c.transferWalletHint).toBe('BDO')
    expect(c.category).toBe('Transfer')
  })

  it('detects InstaPay out as transfer', () => {
    const c = classifyRow(row('InstaPay transfer to 09171234567', '500.00', ''))
    expect(c.kind).toBe('transfer_out')
  })

  it('prefers a known owned wallet name over generic hints', () => {
    const c = classifyRow(row('Transfer to MyMayaWallet', '500.00', ''), ['MyMayaWallet'])
    expect(c.kind).toBe('transfer_out')
    expect(c.transferWalletHint).toBe('MyMayaWallet')
  })

  it('defaults unknown debits to expense', () => {
    const c = classifyRow(row('QR Payment 7-Eleven', '120.00', ''))
    expect(c.kind).toBe('expense')
  })
})
