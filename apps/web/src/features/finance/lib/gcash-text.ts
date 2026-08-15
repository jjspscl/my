/**
 * Pure text-layout logic for GCash statement parsing.
 *
 * The GCash statement renders each transaction as one visual line of PDF text
 * items: date/time, description, reference number, and numeric columns. Text
 * items carry (x, y) from the PDF transformation matrix.
 *
 * IMPORTANT: GCash exports differ wildly in granularity. Some PDFs emit whole
 * words ("Date and Time") as a single item; the reference sample emits ONE
 * item PER CHARACTER (5,565 non-empty items on page 1). Everything here works
 * on geometrically reconstructed SEGMENTS, never on raw item strings, so both
 * layouts parse identically.
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

/**
 * A run of text items that belong together on one visual line: characters
 * with small x-gaps merge into one segment; word-sized gaps split segments.
 * `text` preserves explicit spaces found in the PDF.
 */
export interface TextSegment {
  text: string
  x: number
  end: number
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

/** Max x-gap (PDF units) between items that still belong to one word/segment. */
const SEGMENT_GAP = 6

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

/**
 * Split a line's items into geometric segments. Adjacent items (per-character
 * PDFs) merge; items separated by a word-sized gap split — which is also what
 * happens naturally for whole-word PDFs.
 */
export function splitSegments(line: GcashLine, gapThreshold = SEGMENT_GAP): TextSegment[] {
  const segments: TextSegment[] = []
  for (const item of line.items) {
    if (item.str === '') continue
    const last = segments[segments.length - 1]
    if (last && item.x - last.end <= gapThreshold) {
      last.text += item.str
      last.end = Math.max(last.end, item.x + item.width)
    } else {
      segments.push({ text: item.str, x: item.x, end: item.x + item.width })
    }
  }
  return segments
}

/** Join a line's segments into display text (explicit spaces preserved). */
export function segmentsText(segments: TextSegment[]): string {
  return segments.map((s) => s.text).join(' ').replace(/\s{2,}/g, ' ').trim()
}

/** Find the numeric column layout from the header line, if present. */
export function findColumns(lines: GcashLine[]): GcashColumns | null {
  for (const line of lines) {
    const segments = splitSegments(line)
    const texts = segments.map((s) => s.text)
    const has = (t: string) => texts.some((x) => x.includes(t))
    if (!has('Date and Time') || !has('Reference') || !has('Debit') || !has('Credit') || !has('Balance')) {
      continue
    }

    const seg = (t: string) => segments.find((s) => s.text.includes(t))
    const debit = seg('Debit')
    const credit = seg('Credit')
    const balance = seg('Balance')
    const reference = seg('Reference')
    if (!debit || !credit || !balance || !reference) continue

    // Debit values are right-aligned: the first digit can sit LEFT of the
    // "Debit" label. The numeric zone starts between the reference column and
    // the debit column, not at the debit label itself.
    const numericStart = (reference.end + debit.x) / 2
    const mid = (a: number, b: number) => (a + b) / 2
    return {
      numericStart,
      debit: { min: numericStart, max: mid(debit.x, credit.x) },
      credit: { min: mid(debit.x, credit.x), max: mid(credit.x, balance.x) },
      balance: { min: mid(credit.x, balance.x), max: Number.POSITIVE_INFINITY },
    }
  }
  return null
}

/** Map a segment's x to the numeric column it falls into. */
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
 * time are often separate PDF text items (or even characters), so the
 * left-of-columns text is joined first and matched as one string.
 */
export function assembleRows(lines: GcashLine[], columns: GcashColumns | null): AssembleResult {
  const rows: GcashRow[] = []
  const warnings: GcashWarning[] = []
  let prevBalanceCents: number | null = null

  for (const line of lines) {
    const segments = splitSegments(line)
    const text = segmentsText(segments)
    if (!text || SKIP_RE.test(text)) continue

    const leftSegments = columns
      ? segments.filter((s) => s.x < columns.numericStart)
      : segments
    const leftText = segmentsText(leftSegments)

    const leftMatch = leftText.match(/^(\d{4}-\d{2}-\d{2}\s+\d{1,2}:\d{2}\s+[AP]M)\s+(.+)$/)
    if (!leftMatch) continue
    const dateTime = leftMatch[1]!
    const rest = leftMatch[2]!.trim()

    // Reference: the LAST standalone 13-digit number. A longer run inside a
    // description (e.g. GL1782529307961784) must never be picked — require
    // non-digit neighbors and choose the final candidate. Only the selected
    // reference is stripped from the description.
    let referenceNo = ''
    for (const m of rest.matchAll(/(\d{13})/g)) {
      const before = rest[m.index! - 1] ?? ''
      const after = rest[m.index! + 13] ?? ''
      if (!/\d/.test(before) && !/\d/.test(after)) {
        referenceNo = m[1]!
      }
    }
    const description = referenceNo
      ? rest.replace(referenceNo, ' ').replace(/\s+/g, ' ').trim()
      : rest

    let debit: string
    let credit: string
    let balance: string
    if (columns) {
      // Numeric values are concatenated WITHOUT added spaces: amounts contain
      // explicit separators (commas, dots) already.
      const debitParts: string[] = []
      const creditParts: string[] = []
      const balanceParts: string[] = []
      for (const seg of segments) {
        if (seg.x < columns.numericStart) continue
        const col = numericColumn(columns, seg.x)
        if (col === 'debit') debitParts.push(seg.text)
        else if (col === 'credit') creditParts.push(seg.text)
        else if (col === 'balance') balanceParts.push(seg.text)
      }
      debit = debitParts.join('').trim()
      credit = creditParts.join('').trim()
      balance = balanceParts.join('').trim()
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
