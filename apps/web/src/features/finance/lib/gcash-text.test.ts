import { describe, expect, it } from 'vitest'

import {
  assembleRows,
  findColumns,
  groupLines,
  parseCents,
  type GcashLine,
  type TextItemLike,
} from './gcash-text'

// Build a line from (str, x) pairs at a given y.
function line(y: number, items: Array<[string, number]>): GcashLine {
  return { y, items: items.map(([str, x]) => ({ str, x, y, width: str.length * 5 })) }
}

const item = (str: string, x: number, y: number): TextItemLike => ({ str, x, y, width: str.length * 5 })

// Synthetic GCash-style statement (coordinates approximate a real statement).
const HEADER = line(800, [
  ['Date and Time', 50],
  ['Description', 200],
  ['Reference No', 400],
  ['Debit', 500],
  ['Credit', 560],
  ['Balance', 630],
])

const ROW1 = line(770, [
  ['2026-07-01', 50],
  ['09:30 AM', 130],
  ['Jollibee', 200],
  ['1234567890123', 400],
  ['150.00', 510],
  ['', 0],
  ['5,000.00', 635],
])

const ROW2 = line(745, [
  ['2026-07-02', 50],
  ['06:00 PM', 130],
  ['Received from Juan', 200],
  ['9876543210987', 400],
  ['', 0],
  ['2,500.00', 565],
  ['7,500.00', 635],
])

describe('groupLines', () => {
  it('groups items by y and sorts by x', () => {
    const lines = groupLines([
      item('b', 200, 770),
      item('a', 50, 770),
      item('other', 50, 600),
    ])
    expect(lines).toHaveLength(2)
    const top = lines.find((l) => l.y === 770)!
    expect(top.items.map((i) => i.str)).toEqual(['a', 'b'])
  })
})

describe('findColumns', () => {
  it('detects numeric columns from the header line', () => {
    const cols = findColumns([HEADER])
    expect(cols).not.toBeNull()
    expect(cols!.debit.min).toBe(0)
    expect(cols!.credit.min).toBeGreaterThan(cols!.debit.min)
    expect(cols!.balance.min).toBeGreaterThan(cols!.credit.min)
  })

  it('returns null when no header is present', () => {
    expect(findColumns([ROW1])).toBeNull()
  })
})

describe('assembleRows', () => {
  it('assembles complete rows in place', () => {
    const cols = findColumns([HEADER])!
    const { rows, warnings } = assembleRows([HEADER, ROW1, ROW2], cols)

    expect(rows).toHaveLength(2)
    expect(rows[0]).toEqual({
      dateTime: '2026-07-01 09:30 AM',
      description: 'Jollibee',
      referenceNo: '1234567890123',
      debit: '150.00',
      credit: '',
      balance: '5,000.00',
    })
    expect(rows[1]!.description).toBe('Received from Juan')
    expect(rows[1]!.credit).toBe('2,500.00')
    expect(warnings).toEqual([])
  })

  it('skips summary and repeated header lines', () => {
    const cols = findColumns([HEADER])!
    const summary = line(700, [['ENDING BALANCE', 50], ['7,500.00', 635]])
    const { rows } = assembleRows([HEADER, ROW1, summary, ROW2], cols)
    expect(rows).toHaveLength(2)
  })

  it('warns on rows with both debit and credit', () => {
    const cols = findColumns([HEADER])!
    const bad = line(700, [
      ['2026-07-03', 50],
      ['10:00 AM', 130],
      ['Odd row', 200],
      ['9999999999999', 400],
      ['10.00', 510],
      ['20.00', 565],
      ['4,000.00', 635],
    ])
    const { warnings } = assembleRows([HEADER, bad], cols)
    expect(warnings.some((w) => w.code === 'both-sides')).toBe(true)
  })

  it('detects balance discontinuities', () => {
    const cols = findColumns([HEADER])!
    const broken = line(700, [
      ['2026-07-03', 50],
      ['10:00 AM', 130],
      ['Something', 200],
      ['8888888888888', 400],
      ['10.00', 510],
      ['', 0],
      ['9,000.00', 635],
    ])
    const { warnings } = assembleRows([HEADER, ROW1, broken], cols)
    expect(warnings.some((w) => w.code === 'balance-gap')).toBe(true)
  })

  it('falls back to regex when no columns detected', () => {
    const { rows } = assembleRows([ROW1], null)
    expect(rows).toHaveLength(1)
    expect(rows[0]!.referenceNo).toBe('1234567890123')
    expect(rows[0]!.debit).toBeTruthy()
  })
})

describe('parseCents', () => {
  it('parses commas and decimals to cents', () => {
    expect(parseCents('1,234.56')).toBe(123456)
    expect(parseCents('150.00')).toBe(15000)
    expect(parseCents('-50.25')).toBe(-5025)
    expect(parseCents('')).toBeNull()
    expect(parseCents(undefined)).toBeNull()
    expect(parseCents('abc')).toBeNull()
  })
})
