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

  it('defaults unknown credits to income, not transfer', () => {
    const c = classifyRow(row('Received GCash from BDO', '', '2,000.00'))
    expect(c.kind).toBe('income')
    expect(c.category).toBe('Income')
  })

  it('defaults unknown debits to expense, not transfer', () => {
    const c = classifyRow(row('Transfer from Me to Juan Dela Cruz', '500.00', ''))
    expect(c.kind).toBe('expense')
  })

  it('classifies refunds/reversals as income', () => {
    const c = classifyRow(row('Refund reversal', '', '350.00'))
    expect(c.kind).toBe('income')
    expect(c.category).toBe('Income')
  })

  it('detects transfer out only with an owned wallet match', () => {
    const c = classifyRow(row('BDO Bank Transfer', '10,000.00', ''), ['BDO'])
    expect(c.kind).toBe('transfer_out')
    expect(c.transferWalletHint).toBe('BDO')
    expect(c.category).toBe('Transfer')
  })

  it('does NOT claim a transfer for bank wording alone', () => {
    const c = classifyRow(row('BDO Bank Transfer', '10,000.00', ''))
    expect(c.kind).toBe('expense')
    expect(c.transferWalletHint).toBe('BDO')
  })

  it('detects transfer in from an owned wallet on credit rows', () => {
    const c = classifyRow(row('Received GCash from Maya', '', '500.00'), ['Maya'])
    expect(c.kind).toBe('transfer_in')
    expect(c.category).toBe('Transfer')
  })

  it('prefers a known owned wallet name over generic hints', () => {
    const c = classifyRow(row('Transfer to MyMayaWallet', '500.00', ''), ['MyMayaWallet'])
    expect(c.kind).toBe('transfer_out')
    expect(c.transferWalletHint).toBe('MyMayaWallet')
  })

  it('is case-insensitive when matching owned wallets', () => {
    const c = classifyRow(row('Transfer to mymayawallet', '500.00', ''), ['MyMayaWallet'])
    expect(c.kind).toBe('transfer_out')
  })

  it('refuses ambiguous ownership matches', () => {
    const c = classifyRow(row('Transfer between BDO and Maya', '500.00', ''), ['BDO', 'Maya'])
    expect(c.kind).toBe('expense')
    // BDO stays as a counterparty hint, but never drives the transfer kind.
    expect(c.transferWalletHint).toBe('BDO')
  })

  it('defaults unknown debits to expense', () => {
    const c = classifyRow(row('QR Payment 7-Eleven', '120.00', ''))
    expect(c.kind).toBe('expense')
  })
})
