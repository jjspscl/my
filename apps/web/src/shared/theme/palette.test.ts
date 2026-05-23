import { describe, it, expect } from 'vitest'
import { PALETTE_TOKENS, paletteVar, paletteBgStyle, paletteTextClass, DEFAULT_PALETTE } from './palette'

describe('PALETTE_TOKENS', () => {
  it('has 12 entries', () => {
    expect(PALETTE_TOKENS).toHaveLength(12)
  })

  it('contains all expected tokens', () => {
    expect(PALETTE_TOKENS).toContain('red')
    expect(PALETTE_TOKENS).toContain('blue')
    expect(PALETTE_TOKENS).toContain('green')
    expect(PALETTE_TOKENS).toContain('purple')
    expect(PALETTE_TOKENS).toContain('indigo')
    expect(PALETTE_TOKENS).toContain('teal')
  })
})

describe('paletteVar', () => {
  it('returns CSS variable for green', () => {
    expect(paletteVar('green')).toBe('var(--palette-green)')
  })

  it('returns CSS variable for blue', () => {
    expect(paletteVar('blue')).toBe('var(--palette-blue)')
  })

  it('returns CSS variable for slate', () => {
    expect(paletteVar('slate')).toBe('var(--palette-slate)')
  })
})

describe('paletteBgStyle', () => {
  it('returns inline style for green', () => {
    expect(paletteBgStyle('green')).toEqual({ backgroundColor: 'var(--palette-green)' })
  })

  it('returns inline style for blue', () => {
    expect(paletteBgStyle('blue')).toEqual({ backgroundColor: 'var(--palette-blue)' })
  })
})

describe('paletteTextClass', () => {
  it('returns Tailwind class for red', () => {
    expect(paletteTextClass('red')).toBe('text-[var(--palette-red)]')
  })
})

describe('DEFAULT_PALETTE', () => {
  it('has all tokens as keys', () => {
    for (const token of PALETTE_TOKENS) {
      expect(DEFAULT_PALETTE).toHaveProperty(token)
    }
  })

  it('all values are hex colors starting with #', () => {
    for (const token of PALETTE_TOKENS) {
      expect(DEFAULT_PALETTE[token]).toMatch(/^#[0-9a-fA-F]{6}$/)
    }
  })

  it('green is #22c55e', () => {
    expect(DEFAULT_PALETTE.green).toBe('#22c55e')
  })

  it('blue is #3b82f6', () => {
    expect(DEFAULT_PALETTE.blue).toBe('#3b82f6')
  })
})