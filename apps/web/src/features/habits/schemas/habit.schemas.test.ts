import { describe, it, expect } from 'vitest'
import {
  HabitSchema,
  CreateHabitSchema,
  PaletteTokenSchema,
  HabitFrequencySchema,
  CompletionsMapSchema,
  ToggleResponseSchema,
} from './habit.schemas'

describe('PaletteTokenSchema', () => {
  it('accepts green', () => {
    expect(PaletteTokenSchema.parse('green')).toBe('green')
  })

  it('accepts all 12 tokens', () => {
    const tokens = ['red', 'orange', 'amber', 'yellow', 'green', 'teal', 'cyan', 'blue', 'indigo', 'purple', 'pink', 'slate']
    for (const t of tokens) {
      expect(PaletteTokenSchema.parse(t)).toBe(t)
    }
  })

  it('rejects unknown color', () => {
    const result = PaletteTokenSchema.safeParse('brown')
    expect(result.success).toBe(false)
  })
})

describe('HabitFrequencySchema', () => {
  it('accepts daily', () => {
    expect(HabitFrequencySchema.parse('daily')).toBe('daily')
  })

  it('accepts weekly', () => {
    expect(HabitFrequencySchema.parse('weekly')).toBe('weekly')
  })

  it('rejects unknown frequency', () => {
    const result = HabitFrequencySchema.safeParse('monthly')
    expect(result.success).toBe(false)
  })
})

describe('CreateHabitSchema', () => {
  it('accepts valid input', () => {
    const result = CreateHabitSchema.safeParse({
      name: 'Exercise',
      color: 'green',
      frequency: 'daily',
      targetPerWeek: 7,
    })
    expect(result.success).toBe(true)
  })

  it('rejects empty name', () => {
    const result = CreateHabitSchema.safeParse({
      name: '',
      color: 'blue',
      frequency: 'daily',
    })
    expect(result.success).toBe(false)
  })

  it('rejects missing name', () => {
    const result = CreateHabitSchema.safeParse({
      color: 'blue',
      frequency: 'daily',
    })
    expect(result.success).toBe(false)
  })

  it('defaults color to blue', () => {
    const input = CreateHabitSchema.parse({
      name: 'Test',
      frequency: 'daily',
    })
    expect(input.color).toBe('blue')
  })

  it('defaults frequency to daily', () => {
    const input = CreateHabitSchema.parse({
      name: 'Test',
    })
    expect(input.frequency).toBe('daily')
  })

  it('defaults targetPerWeek to 1', () => {
    const input = CreateHabitSchema.parse({
      name: 'Test',
    })
    expect(input.targetPerWeek).toBe(1)
  })

  it('default targetPerWeek applies when not provided', () => {
    const input = CreateHabitSchema.parse({
      name: 'Test',
      frequency: 'weekly',
    })
    expect(input.targetPerWeek).toBe(1)
  })
})

describe('HabitSchema', () => {
  it('parses valid habit', () => {
    const habit = HabitSchema.parse({
      id: 'h-001',
      name: 'Exercise',
      color: 'green',
      frequency: 'daily',
      targetPerWeek: 7,
      archived: false,
      createdAt: '2026-01-01T00:00:00Z',
    })
    expect(habit.id).toBe('h-001')
    expect(habit.color).toBe('green')
    expect(habit.archived).toBe(false)
  })

  it('parses habit with optional fields', () => {
    const habit = HabitSchema.parse({
      id: 'h-002',
      name: 'Read',
      color: 'blue',
      frequency: 'daily',
      targetPerWeek: 7,
      archived: false,
      createdAt: '2026-01-01T00:00:00Z',
      completedToday: true,
      currentStreak: 5,
    })
    expect(habit.completedToday).toBe(true)
    expect(habit.currentStreak).toBe(5)
  })

  it('completedToday defaults to undefined', () => {
    const habit = HabitSchema.parse({
      id: 'h-003',
      name: 'Meditate',
      color: 'purple',
      frequency: 'daily',
      targetPerWeek: 7,
      archived: false,
      createdAt: '2026-01-01T00:00:00Z',
    })
    expect(habit.completedToday).toBeUndefined()
  })

  it('rejects invalid color', () => {
    const result = HabitSchema.safeParse({
      id: 'h-004',
      name: 'Bad',
      color: 'brown',
      frequency: 'daily',
      targetPerWeek: 1,
      archived: false,
      createdAt: '2026-01-01T00:00:00Z',
    })
    expect(result.success).toBe(false)
  })

  it('rejects archived as string', () => {
    const result = HabitSchema.safeParse({
      id: 'h-005',
      name: 'Test',
      color: 'blue',
      frequency: 'daily',
      targetPerWeek: 1,
      archived: 'true',
      createdAt: '2026-01-01T00:00:00Z',
    })
    expect(result.success).toBe(false)
  })
})

describe('ToggleResponseSchema', () => {
  it('parses completed true', () => {
    const resp = ToggleResponseSchema.parse({ completed: true })
    expect(resp.completed).toBe(true)
  })

  it('parses completed false', () => {
    const resp = ToggleResponseSchema.parse({ completed: false })
    expect(resp.completed).toBe(false)
  })

  it('rejects missing completed', () => {
    const result = ToggleResponseSchema.safeParse({})
    expect(result.success).toBe(false)
  })
})

describe('CompletionsMapSchema', () => {
  it('parses valid completions map', () => {
    const data = CompletionsMapSchema.parse({
      completions: {
        '2026-01-01': ['h-001', 'h-002'],
        '2026-01-02': ['h-001'],
      },
      totalHabits: 5,
    })
    expect(data.completions['2026-01-01']).toHaveLength(2)
    expect(data.completions['2026-01-02']).toHaveLength(1)
    expect(data.totalHabits).toBe(5)
  })

  it('parses empty completions', () => {
    const data = CompletionsMapSchema.parse({
      completions: {},
      totalHabits: 3,
    })
    expect(Object.keys(data.completions)).toHaveLength(0)
    expect(data.totalHabits).toBe(3)
  })

  it('rejects missing completions', () => {
    const result = CompletionsMapSchema.safeParse({
      totalHabits: 5,
    })
    expect(result.success).toBe(false)
  })

  it('accepts negative totalHabits (schema allows int, no positive constraint)', () => {
    const result = CompletionsMapSchema.safeParse({
      completions: {},
      totalHabits: -1,
    })
    expect(result.success).toBe(true) // schema only enforces int, not positive
  })
})