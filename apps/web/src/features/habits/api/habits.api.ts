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

/** Local calendar date (YYYY-MM-DD). Frozen at queue time for offline
 * replays — a queued "today" must not drift when drained days later. */
export function todayLocal(): string {
  const d = new Date()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${mm}-${dd}`
}

export async function toggleHabit(habitId: string, date?: string, completed?: boolean) {
  const body: Record<string, string | boolean> = {}
  if (date) body.date = date
  // Explicit set-state: idempotent server-side, replay-safe from the queue.
  if (completed !== undefined) body.completed = completed
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