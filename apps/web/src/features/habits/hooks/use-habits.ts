import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { habitKeys } from '../api/habits.keys'
import {
  archiveHabit as archiveHabitApi,
  createHabit as createHabitApi,
  getCompletions as getCompletionsApi,
  getCompletionsMap as getCompletionsMapApi,
  listHabits as listHabitsApi,
  toggleHabit as toggleHabitApi,
} from '../api/habits.api'
import type { CreateHabit } from '../schemas/habit.schemas'

export function useHabits() {
  return useQuery({
    queryKey: habitKeys.list(),
    queryFn: listHabitsApi,
    staleTime: 1000 * 30,
  })
}

export function useCreateHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateHabit) => createHabitApi(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useToggleHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ habitId, date }: { habitId: string; date?: string }) =>
      toggleHabitApi(habitId, date),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useArchiveHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (habitId: string) => archiveHabitApi(habitId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useCompletions(habitId: string, from?: string, to?: string) {
  return useQuery({
    queryKey: [...habitKeys.completions(habitId), from, to],
    queryFn: () => getCompletionsApi(habitId, from, to),
    staleTime: 1000 * 60,
  })
}

export function useCompletionsMap(from?: string, to?: string) {
  return useQuery({
    queryKey: habitKeys.completionsMap(from, to),
    queryFn: () => getCompletionsMapApi(from, to),
    staleTime: 1000 * 60,
  })
}