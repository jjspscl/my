export const PALETTE_TOKENS = [
  'red',
  'orange',
  'amber',
  'yellow',
  'green',
  'teal',
  'cyan',
  'blue',
  'indigo',
  'purple',
  'pink',
  'slate',
] as const

export type PaletteToken = (typeof PALETTE_TOKENS)[number]

export const DEFAULT_PALETTE: Record<PaletteToken, string> = {
  red: '#ef4444',
  orange: '#f97316',
  amber: '#f59e0b',
  yellow: '#eab308',
  green: '#22c55e',
  teal: '#14b8a6',
  cyan: '#06b6d4',
  blue: '#3b82f6',
  indigo: '#6366f1',
  purple: '#a855f7',
  pink: '#ec4899',
  slate: '#64748b',
}

/** Returns CSS variable reference for a palette token. */
export function paletteVar(token: PaletteToken): string {
  return `var(--palette-${token})`
}

/** Returns Tailwind arbitrary color class for a palette token. */
export function paletteBgClass(token: PaletteToken): string {
  return `bg-[var(--palette-${token})]`
}

/** Returns inline style object for background color. */
export function paletteBgStyle(token: PaletteToken): React.CSSProperties {
  return { backgroundColor: `var(--palette-${token})` }
}

export function paletteTextClass(token: PaletteToken): string {
  return `text-[var(--palette-${token})]`
}