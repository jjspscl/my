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
    // Numeric zone starts between the reference column end and the Debit
    // label (right-aligned amounts may start left of the label).
    expect(cols!.debit.min).toBe(480)
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

// ---- character-level PDF fixtures (one TextItem per character) ----

const CHAR_W = 4.5 // per-character advance

function charItems(text: string, startX: number, y: number): TextItemLike[] {
  const items: TextItemLike[] = []
  let x = startX
  for (const ch of text) {
    items.push({ str: ch, x, y, width: ch === ' ' ? CHAR_W : CHAR_W })
    x += CHAR_W
  }
  return items
}

// Header rendered one character at a time, 30 units between words.
function charLine(y: number, words: Array<[string, number]>): GcashLine {
  const items: TextItemLike[] = []
  for (const [word, x] of words) {
    items.push(...charItems(word, x, y))
  }
  return { y, items }
}

const CHAR_HEADER = charLine(800, [
  ['Date and Time', 50],
  ['Description', 211],
  ['Reference No.', 341],
  ['Debit', 427],
  ['Credit', 476],
  ['Balance', 524],
])

function charRow(
  y: number,
  date: string,
  desc: string,
  ref: string,
  debit: string,
  credit: string,
  balance: string,
): GcashLine {
  const items: TextItemLike[] = []
  items.push(...charItems(date + ' ', 50, y))
  items.push(...charItems(desc + ' ', 126, y))
  items.push(...charItems(ref + ' ', 329, y))
  // Right-aligned amounts: digits may start left of the column label.
  if (debit) items.push(...charItems(debit, 421 - (debit.length - 6) * CHAR_W, y))
  if (credit) items.push(...charItems(credit, 468 - (credit.length - 6) * CHAR_W, y))
  items.push(...charItems(balance, 521 - (balance.length - 6) * CHAR_W, y))
  return { y, items }
}

describe('character-level PDFs (one item per char)', () => {
  it('reconstructs the header and detects columns', () => {
    const cols = findColumns([CHAR_HEADER])
    expect(cols).not.toBeNull()
    expect(cols!.numericStart).toBeGreaterThan(380) // between Reference end and Debit
    expect(cols!.numericStart).toBeLessThan(427)
  })

  it('assembles rows with right-aligned debits starting left of the label', () => {
    const cols = findColumns([CHAR_HEADER])!
    const row = charRow(770, '2026-07-01 09:30 AM', 'Jollibee', '1234567890123', '1510.11', '', '38808.39')

    const { rows, warnings } = assembleRows([CHAR_HEADER, row], cols)
    expect(rows).toHaveLength(1)
    expect(rows[0]).toEqual({
      dateTime: '2026-07-01 09:30 AM',
      description: 'Jollibee',
      referenceNo: '1234567890123',
      debit: '1510.11',
      credit: '',
      balance: '38808.39',
    })
    expect(warnings).toEqual([])
  })

  it('parses credit rows too', () => {
    const cols = findColumns([CHAR_HEADER])!
    const row = charRow(770, '2026-07-02 06:00 PM', 'Received from Juan', '9876543210987', '', '2500.00', '7500.00')

    const { rows } = assembleRows([CHAR_HEADER, row], cols)
    expect(rows).toHaveLength(1)
    expect(rows[0]!.credit).toBe('2500.00')
    expect(rows[0]!.debit).toBe('')
  })

  it('picks the standalone 13-digit reference, not a suffix of a longer number', () => {
    const cols = findColumns([CHAR_HEADER])!
    // GL1782529307961784 contains a 13-digit run with a digit after it — must
    // be rejected. The real reference follows.
    const row = charRow(770, '2026-06-27 11:01 AM', 'GLoan Repayment GL1782529307961784', '5042278509918', '1510.11', '', '38808.39')

    const { rows, warnings } = assembleRows([CHAR_HEADER, row], cols)
    expect(rows).toHaveLength(1)
    expect(rows[0]!.referenceNo).toBe('5042278509918')
    expect(rows[0]!.description).toContain('GL1782529307961784')
    expect(warnings).toEqual([])
  })
})
