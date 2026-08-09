/**
 * Shared formatting for finance analytics. Amounts are minor units (cents)
 * everywhere in the API; format at the edge only.
 */

/** Formats minor-unit cents as a currency string, e.g. ₱1,234.56. */
export function formatCents(cents: number, currency = 'PHP'): string {
  return new Intl.NumberFormat('en-PH', {
    style: 'currency',
    currency,
    maximumFractionDigits: 2,
  }).format(cents / 100)
}

/** Formats a YYYY-MM month as a short label, e.g. "Jul 2026". */
export function formatMonth(month: string): string {
  const [year, monthNum] = month.split('-')
  if (!year || !monthNum) return month
  const date = new Date(Number(year), Number(monthNum) - 1, 1)
  return date.toLocaleString('en-PH', { month: 'short', year: 'numeric' })
}

/** Current month as YYYY-MM in the user's local timezone. */
export function currentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}