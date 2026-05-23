import { z } from 'zod'

export const PaletteTokenSchema = z.enum([
  'red','orange','amber','yellow','green','teal','cyan','blue','indigo','purple','pink','slate',
])
export type PaletteToken = z.infer<typeof PaletteTokenSchema>

export const HabitFrequencySchema = z.enum(['daily', 'weekly'])
export type HabitFrequency = z.infer<typeof HabitFrequencySchema>

export const CreateHabitSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  color: PaletteTokenSchema.default('blue'),
  frequency: HabitFrequencySchema.default('daily'),
  targetPerWeek: z.number().int().min(1).default(1),
})
export type CreateHabit = z.infer<typeof CreateHabitSchema>

export const HabitSchema = z.object({
  id: z.string(),
  name: z.string(),
  color: PaletteTokenSchema,
  frequency: HabitFrequencySchema,
  targetPerWeek: z.number().int(),
  archived: z.boolean(),
  createdAt: z.string(),
  completedToday: z.boolean().optional(),
  currentStreak: z.number().int().optional(),
})
export type Habit = z.infer<typeof HabitSchema>

export const CompletionSchema = z.object({
  id: z.string(),
  completedDate: z.string(),
})
export type Completion = z.infer<typeof CompletionSchema>

export const ToggleResponseSchema = z.object({
  completed: z.boolean(),
})
export type ToggleResponse = z.infer<typeof ToggleResponseSchema>

export const ApiOKResponseSchema = z.object({
  ok: z.boolean().optional(),
  error: z.string().optional(),
})

export const CompletionsMapSchema = z.object({
  completions: z.record(z.string(), z.array(z.string())),
  totalHabits: z.number().int(),
})
export type CompletionsMap = z.infer<typeof CompletionsMapSchema>