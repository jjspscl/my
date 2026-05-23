import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import {
  ApiOKResponseSchema,
  type CreateHabit,
  CreateHabitSchema,
  HabitSchema,
  ToggleResponseSchema,
  CompletionsMapSchema,
} from '../schemas/habit.schemas'

const HabitsListDataSchema = z.object({ data: z.array(HabitSchema) })
const HabitDataSchema = z.object({ data: HabitSchema })
const ToggleDataSchema = z.object({ data: ToggleResponseSchema })
const CompletionsDataSchema = z.object({
  data: z.array(z.object({ id: z.string(), completedDate: z.string() })),
})

export async function createHabit(data: CreateHabit) {
  const parsed = CreateHabitSchema.parse(data)
  return apiClient('/api/v1/habits', HabitDataSchema, {
    method: 'POST',
    body: JSON.stringify(parsed),
  })
}

export async function listHabits() {
  const res = await apiClient('/api/v1/habits', HabitsListDataSchema)
  return res.data
}

export async function toggleHabit(habitId: string, date?: string) {
  const body: Record<string, string> = {}
  if (date) body.date = date
  const res = await apiClient(`/api/v1/habits/${habitId}/toggle`, ToggleDataSchema, {
    method: 'POST',
    body: JSON.stringify(body),
  })
  return res.data
}

export async function archiveHabit(habitId: string) {
  return apiClient(`/api/v1/habits/${habitId}/archive`, ApiOKResponseSchema, {
    method: 'PATCH',
  })
}

export async function getCompletions(habitId: string, from?: string, to?: string) {
  const params = new URLSearchParams()
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  const qs = params.toString()
  const res = await apiClient(
    `/api/v1/habits/${habitId}/completions${qs ? `?${qs}` : ''}`,
    CompletionsDataSchema,
  )
  return res.data
}

export async function getCompletionsMap(from?: string, to?: string) {
  const params = new URLSearchParams()
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  const qs = params.toString()
  const res = await apiClient(
    `/api/v1/habits/completions${qs ? `?${qs}` : ''}`,
    z.object({ data: CompletionsMapSchema }),
  )
  return res.data
}