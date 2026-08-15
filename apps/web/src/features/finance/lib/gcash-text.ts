/**
 * Pure text-layout logic for GCash statement parsing.
 *
 * The GCash statement renders each transaction as one visual line of PDF text
 * items: date/time, description, reference number, and numeric columns. Text
 * items carry (x, y) from the PDF transformation matrix.
 *
 * This module turns raw text items into transaction rows WITHOUT PDF.js so it
 * can be unit-tested against synthetic fixtures. The pdf.js loader in
 * gcash-pdf-parser.ts only extracts text items per page and feeds them here.
 */

export interface TextItemLike {
  str: string
  x: number
  y: number
  width: number
}

export interface GcashLine {
  y: number
  items: TextItemLike[]
}

export interface GcashColumns {
  /** x boundary between left (date/desc/ref) and numeric columns */
  numericStart: number
  debit: { min: number; max: number }
  credit: { min: number; max: number }
  balance: { min: number; max: number }
}

export interface GcashRow {
  dateTime: string // "YYYY-MM-DD HH:MM AM/PM"
  description: string
  referenceNo: string
  debit: string // raw, may be empty
  credit: string // raw, may be empty
  balance: string // raw, may be empty
}

export type WarningCode =
  | 'both-sides'
  | 'no-side'
  | 'bad-reference'
  | 'unparseable-date'
  | 'balance-gap'

export interface GcashWarning {
  rowIndex?: number
  code: WarningCode
  message: string
}

const SKIP_RE = /^(STARTING BALANCE|ENDING BALANCE|TOTAL DEBIT|TOTAL CREDIT|Balance|Date and Time)/i

const ROW_Y_TOLERANCE = 2.5

/** Group text items into visual lines by y-coordinate. */
export function groupLines(items: TextItemLike[]): GcashLine[] {
  const sorted = [...items].sort((a, b) => b.y - a.y)
  const lines: GcashLine[] = []

  for (const item of sorted) {
    let added = false
    for (const line of lines) {
      if (Math.abs(item.y - line.y) <= ROW_Y_TOLERANCE) {
        line.items.push(item)
        added = true
        break
      }
    }
    if (!added) {
      lines.push({ y: item.y, items: [item] })
    }
  }

  for (const line of lines) {
    line.items.sort((a, b) => a.x - b.x)
  }
  return lines
}

/** Find the numeric column layout from the header line, if present. */
export function findColumns(lines: GcashLine[]): GcashColumns | null {
  for (const line of lines) {
    const texts = line.items.map((i) => i.str)
    if (!texts.some((t) => t.includes('Date and Time'))) continue
    if (!texts.some((t) => t.includes('Reference'))) continue

    const get = (name: string): TextItemLike | undefined =>
      line.items.find((i) => i.str.includes(name))

    const debit = get('Debit')
    const credit = get('Credit')
    const balance = get('Balance')
    if (!debit || !credit || !balance) continue

    const numericStart = debit.x
    const mid = (a: number, b: number) => a + (b - a) / 2
    return {
      numericStart,
      debit: { min: 0, max: mid(debit.x, credit.x) },
      credit: { min: mid(debit.x, credit.x), max: mid(credit.x, balance.x) },
      balance: { min: mid(credit.x, balance.x), max: Number.POSITIVE_INFINITY },
    }
  }
  return null
}

/** Map a text item to the numeric column it falls into. */
function numericColumn(cols: GcashColumns, x: number): 'debit' | 'credit' | 'balance' | null {
  if (x >= cols.balance.min) return 'balance'
  if (x >= cols.credit.min) return 'credit'
  if (x >= cols.debit.min) return 'debit'
  return null
}

export interface AssembleResult {
  rows: GcashRow[]
  warnings: GcashWarning[]
}

/**
 * Assemble transaction rows from lines + detected columns.
 *
 * Every row is assembled in place — description, reference, and amounts come
 * from the same visual line, never zipped from separate arrays. The date and
 * time are often separate PDF text items, so the left-of-columns text is
 * joined first and matched as one "date time rest" string.
 */
export function assembleRows(lines: GcashLine[], columns: GcashColumns | null): AssembleResult {
  const rows: GcashRow[] = []
  const warnings: GcashWarning[] = []
  let prevBalanceCents: number | null = null

  for (const line of lines) {
    const text = line.items.map((i) => i.str).join(' ').trim()
    if (!text || SKIP_RE.test(text)) continue

    const leftItems = columns ? line.items.filter((i) => i.x < columns.numericStart) : line.items
    const leftText = leftItems.map((i) => i.str).join(' ').trim()

    const leftMatch = leftText.match(/^(\d{4}-\d{2}-\d{2}\s+\d{1,2}:\d{2}\s+[AP]M)\s+(.+)$/)
    if (!leftMatch) continue
    const dateTime = leftMatch[1]!
    let rest = leftMatch[2]!.trim()

    // Reference: 13-digit number in the left text (followed by space or end).
    let referenceNo = ''
    const refMatch = rest.match(/(\d{13})(?=\s|$)/)
    if (refMatch) {
      referenceNo = refMatch[1]!
      rest = rest.replace(/(\d{13})(?=\s|$)/, '').trim()
    }
    const description = rest.replace(/\s{2,}/g, ' ')

    let debit: string
    let credit: string
    let balance: string
    if (columns) {
      const debitParts: string[] = []
      const creditParts: string[] = []
      const balanceParts: string[] = []
      for (const item of line.items) {
        if (item.x < columns.numericStart) continue
        const col = numericColumn(columns, item.x)
        if (col === 'debit') debitParts.push(item.str)
        else if (col === 'credit') creditParts.push(item.str)
        else if (col === 'balance') balanceParts.push(item.str)
      }
      debit = debitParts.join(' ').trim()
      credit = creditParts.join(' ').trim()
      balance = balanceParts.join(' ').trim()
    } else {
      // No header detected: text joins left-to-right, so the first decimal is
      // the debit column, then credit, then balance.
      const amounts = text.match(/\d[\d,]*\.\d{2}/g) ?? []
      debit = amounts[0] ?? ''
      credit = amounts[1] ?? ''
      balance = amounts[2] ?? ''
    }

    if (!referenceNo) {
      warnings.push({ code: 'bad-reference', message: `Row ${dateTime} has no 13-digit reference` })
    }
    if (debit && credit) {
      warnings.push({ code: 'both-sides', message: `Row ${dateTime} has both debit and credit` })
    }
    if (!debit && !credit) {
      warnings.push({ code: 'no-side', message: `Row ${dateTime} has neither debit nor credit` })
    }

    rows.push({ dateTime, description, referenceNo, debit, credit, balance })

    // Balance continuity check (in cents) — catches rows the parser may have
    // merged or dropped.
    const debitCents = parseCents(debit)
    const creditCents = parseCents(credit)
    const balanceCents = parseCents(balance)
    if (prevBalanceCents !== null && balanceCents !== null && (debitCents !== null || creditCents !== null)) {
      const expected = prevBalanceCents - (debitCents ?? 0) + (creditCents ?? 0)
      if (Math.abs(expected - balanceCents) > 1) {
        warnings.push({
          rowIndex: rows.length - 1,
          code: 'balance-gap',
          message: `Balance discontinuity at ${dateTime}: expected ${expected / 100}, got ${balanceCents / 100}`,
        })
      }
    }
    if (balanceCents !== null) prevBalanceCents = balanceCents
  }

  return { rows, warnings }
}

/** Parse "1,234.56" or "-1,234.56" into integer cents; null when absent. */
export function parseCents(raw: string | undefined | null): number | null {
  if (!raw) return null
  const cleaned = raw.replace(/[,\s]/g, '')
  const n = Number(cleaned)
  if (!Number.isFinite(n)) return null
  return Math.round(n * 100)
}

/** Format cents back to "1,234.56". */
export function formatCents(cents: number): string {
  return (cents / 100).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}
