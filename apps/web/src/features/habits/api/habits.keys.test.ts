import { describe, it, expect } from 'vitest'
import { habitKeys } from './habits.keys'

describe('habitKeys', () => {
  it('all returns base key', () => {
    expect(habitKeys.all).toEqual(['habits'])
  })

  it('list returns key with list suffix', () => {
    expect(habitKeys.list()).toEqual(['habits', 'list'])
  })

  it('completions returns key with habit ID', () => {
    expect(habitKeys.completions('h-001')).toEqual(['habits', 'completions', 'h-001'])
  })

  it('completionsMap returns key with date params', () => {
    expect(habitKeys.completionsMap('2026-01-01', '2026-05-23')).toEqual(['habits', 'completions-map', '2026-01-01', '2026-05-23'])
  })

  it('completionsMap without dates returns key with undefined params', () => {
    const keys = habitKeys.completionsMap()
    expect(keys).toEqual(['habits', 'completions-map', undefined, undefined])
  })

  it('list and completions produce different keys', () => {
    expect(habitKeys.list()).not.toEqual(habitKeys.completions('h-001'))
  })

  it('completions for different IDs produce different keys', () => {
    expect(habitKeys.completions('h-001')).not.toEqual(habitKeys.completions('h-002'))
  })
})