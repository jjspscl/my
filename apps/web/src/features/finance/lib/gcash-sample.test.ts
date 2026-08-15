import { readFileSync, readdirSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import { assembleRows, findColumns, groupLines } from './gcash-text'

// Optional local validation against the maintainer's real GCash statement.
// The file lives in the gitignored data/ directory, so CI (and any checkout
// without the sample) skips these assertions. Only counts and layout facts
// are asserted — never transaction values.
// vite-node rewrites module URLs to /@fs/<abs>; strip it to get the real path.
const dataDir = new URL('../../../../../../data/', import.meta.url).pathname.replace(/^\/@fs/, '')
const pdfs = (() => {
  try {
    return readdirSync(dataDir).filter((f) => f.endsWith('.pdf'))
  } catch {
    return []
  }
})()

function extractItemsFromTextContent(page: unknown) {
  // Mirror of the loader mapping in gcash-pdf-parser.ts.
  const content = page as {
    items: Array<{ str: string; transform: number[]; width: number }>
  }
  return content.items
    .filter((item) => typeof item.str === 'string' && item.str !== '')
    .map((item) => ({
      str: item.str,
      x: item.transform[4]!,
      y: item.transform[5]!,
      width: item.width,
    }))
}

describe('real GCash sample (skipped when data/ PDFs absent)', () => {
  it.skipIf(pdfs.length === 0)('parses the sample statement layout', async () => {
    const pdfPath = `${dataDir}/${pdfs[0]}`
    const pdfjs = await import('pdfjs-dist/legacy/build/pdf.mjs')
    const doc = await pdfjs.getDocument({ data: new Uint8Array(readFileSync(pdfPath)) }).promise

    let columns: ReturnType<typeof findColumns> = null
    let sawHeader = false
    let rows = 0
    let pages = 0

    try {
      for (let page = 1; page <= doc.numPages; page++) {
        pages++
        const pdfPage = await doc.getPage(page)
        const content = await pdfPage.getTextContent()
        const lines = groupLines(extractItemsFromTextContent(content))
        if (!sawHeader) {
          const detected = findColumns(lines)
          if (detected) {
            columns = detected
            sawHeader = true
          }
        }
        rows += assembleRows(lines, columns).rows.length
      }
    } finally {
      await doc.destroy()
    }

    expect(sawHeader).toBe(true)
    expect(pages).toBeGreaterThan(1)
    // Reference sample: 4 pages, 61/67/68/11 = 207 transaction rows.
    expect(rows).toBe(207)
    expect(columns!.numericStart).toBeGreaterThan(380)
    expect(columns!.numericStart).toBeLessThan(430)
  })
})
