/**
 * GCash statement PDF parser.
 *
 * Parsing happens entirely in the browser: the file and its password never
 * leave the device. Text items are extracted with pdf.js and assembled by the
 * pure layout logic in gcash-text.ts (unit-tested against synthetic fixtures).
 */

import { getDocument, GlobalWorkerOptions } from 'pdfjs-dist'
import type { TextItem } from 'pdfjs-dist/types/src/display/api'

import {
  assembleRows,
  findColumns,
  formatCents,
  groupLines,
  parseCents,
  type GcashRow,
  type GcashWarning,
  type TextItemLike,
} from './gcash-text'

let workerConfigured = false

function ensureWorker() {
  if (workerConfigured || typeof window === 'undefined') return
  if ('Worker' in window) {
    GlobalWorkerOptions.workerPort = new Worker(
      new URL('pdfjs-dist/build/pdf.worker.min.mjs', import.meta.url),
      { type: 'module' },
    )
  }
  workerConfigured = true
}

export class GcashParseError extends Error {
  code: 'password' | 'corrupt' | 'unsupported' | 'empty'
  constructor(code: 'password' | 'corrupt' | 'unsupported' | 'empty', message: string) {
    super(message)
    this.code = code
  }
}

export interface GcashParsedStatement {
  rows: GcashRow[]
  totalDebitCents: number
  totalCreditCents: number
  startingBalanceCents: number | null
  endingBalanceCents: number | null
  warnings: GcashWarning[]
}

/**
 * Parse a GCash statement PDF. `password` is optional; GCash statements
 * usually require one.
 */
export async function parseGcashPdf(file: ArrayBuffer, password?: string): Promise<GcashParsedStatement> {
  ensureWorker()

  let doc
  try {
    doc = await getDocument({
      data: new Uint8Array(file),
      password: password ?? '',
    }).promise
  } catch (err) {
    const name = err instanceof Error ? err.name : ''
    if (name === 'PasswordException') {
      throw new GcashParseError('password', 'Invalid password for this statement.')
    }
    throw new GcashParseError('corrupt', `Could not read the PDF: ${err instanceof Error ? err.message : 'unknown error'}`)
  }

  const warnings: GcashWarning[] = []
  const rows: GcashRow[] = []
  let columns: ReturnType<typeof findColumns> = null
  let sawHeader = false

  try {
    for (let page = 1; page <= doc.numPages; page++) {
      const pdfPage = await doc.getPage(page)
      const content = await pdfPage.getTextContent()

      const items: TextItemLike[] = content.items
        .filter((item): item is TextItem => 'str' in item)
        .map((item) => ({
          str: item.str,
          x: item.transform[4],
          y: item.transform[5],
          width: item.width,
        }))

      const lines = groupLines(items)
      if (!sawHeader) {
        const detected = findColumns(lines)
        if (detected) {
          columns = detected
          sawHeader = true
        }
      }

      const assembled = assembleRows(lines, columns)
      rows.push(...assembled.rows)
      warnings.push(...assembled.warnings)
    }
  } finally {
    await doc.destroy()
  }

  if (rows.length === 0) {
    if (!sawHeader) {
      throw new GcashParseError(
        'unsupported',
        'No GCash statement layout detected. Is this a GCash Transaction History PDF?',
      )
    }
    throw new GcashParseError('empty', 'No transactions found in this statement.')
  }

  // Statement totals + running balance.
  let totalDebitCents = 0
  let totalCreditCents = 0
  let startingBalanceCents: number | null = null
  let endingBalanceCents: number | null = null

  for (const row of rows) {
    const debit = parseCents(row.debit)
    const credit = parseCents(row.credit)
    if (debit !== null) totalDebitCents += debit
    if (credit !== null) totalCreditCents += credit
  }

  const first = rows[0]
  const last = rows[rows.length - 1]
  const firstBalance = first ? parseCents(first.balance) : null
  const lastBalance = last ? parseCents(last.balance) : null
  if (firstBalance !== null && first) {
    startingBalanceCents = firstBalance - (parseCents(first.credit) ?? 0) + (parseCents(first.debit) ?? 0)
  }
  if (lastBalance !== null) {
    endingBalanceCents = lastBalance
    const expectedEnding =
      (startingBalanceCents ?? 0) + totalCreditCents - totalDebitCents
    if (startingBalanceCents !== null && Math.abs(expectedEnding - endingBalanceCents) > 1) {
      warnings.push({
        code: 'balance-gap',
        message: `Statement totals do not reconcile: expected ending ${formatCents(expectedEnding)}, got ${formatCents(endingBalanceCents)}.`,
      })
    }
  }

  return {
    rows,
    totalDebitCents,
    totalCreditCents,
    startingBalanceCents,
    endingBalanceCents,
    warnings,
  }
}
