import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Formats a Date as YYYY-MM-DD in the user's LOCAL timezone. The backend
 * treats all dates as belonging to the user's financial calendar
 * (MY_TIMEZONE), so converting with toISOString() — which shifts to UTC —
 * silently moves transactions across days. Never use toISOString for dates.
 */
export function toLocalDateStr(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/** Today's date as YYYY-MM-DD in the user's local timezone. */
export function todayLocalStr(): string {
  return toLocalDateStr(new Date())
}
