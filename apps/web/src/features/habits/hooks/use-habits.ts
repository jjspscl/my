import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { habitKeys } from '../api/habits.keys'
import { useNetworkStatus } from '@/shared/sync/network-status'
import { offlineMutate } from '@/shared/sync/offline-mutate'
import {
  archiveHabit as archiveHabitApi,
  createHabit as createHabitApi,
  getCompletions as getCompletionsApi,
  getCompletionsMap as getCompletionsMapApi,
  listHabits as listHabitsApi,
  toggleHabit as toggleHabitApi,
} from '../api/habits.api'
import type { CreateHabit, Habit } from '../schemas/habit.schemas'

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
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Could not create the habit.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useToggleHabit() {
  const queryClient = useQueryClient()

  return useMutation<
    { completed: boolean } | Response | 'queued',
    Error,
    { habitId: string; date?: string; completed?: boolean },
    { prev: Habit[] | undefined }
  >({
    mutationFn: ({
      habitId,
      date,
      completed,
    }: {
      habitId: string
      date?: string
      completed?: boolean
    }) => {
      const body: Record<string, string | boolean> = {}
      if (date) body.date = date
      if (completed !== undefined) body.completed = completed
      if (useNetworkStatus.getState().isOnline) {
        return toggleHabitApi(habitId, date, completed)
      }
      // Offline: queue an explicit set-state. The date is frozen by the
      // caller so a later replay completes the right day, and the explicit
      // completed flag makes the replay idempotent (a flip would un-check).
      return offlineMutate(`/api/v1/habits/${habitId}/toggle`, {
        method: 'POST',
        body,
      })
    },
    onMutate: async ({ habitId }): Promise<{ prev: Habit[] | undefined }> => {
      await queryClient.cancelQueries({ queryKey: habitKeys.list() })
      const prev = queryClient.getQueryData<Habit[]>(habitKeys.list())

      // Optimistically flip completedToday
      queryClient.setQueryData<Habit[]>(habitKeys.list(), (old) =>
        old
          ? old.map((h) =>
              h.id === habitId
                ? {
                    ...h,
                    completedToday: !h.completedToday,
                    currentStreak: h.completedToday
                      ? Math.max(0, (h.currentStreak ?? 0) - 1)
                      : (h.currentStreak ?? 0) + 1,
                  }
                : h,
            )
          : old,
      )

      return { prev }
    },
    onError: (err, _vars, ctx) => {
      if (ctx?.prev) {
        queryClient.setQueryData(habitKeys.list(), ctx.prev)
      }
      toast.error(err instanceof Error ? err.message : 'Could not update the habit.')
    },
    onSuccess: (data) => {
      // 'queued' = offlineMutate parked the change in the queue — make the
      // state visible instead of looking like a silent success.
      if (data === 'queued') {
        toast.info('Saved offline — will sync when you reconnect')
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useArchiveHabit() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (habitId: string) => archiveHabitApi(habitId),
    onMutate: async (habitId) => {
      await queryClient.cancelQueries({ queryKey: habitKeys.list() })
      const prev = queryClient.getQueryData<Habit[]>(habitKeys.list())

      // Optimistically remove from list
      queryClient.setQueryData<Habit[]>(habitKeys.list(), (old) =>
        old ? old.filter((h) => h.id !== habitId) : old,
      )

      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) {
        queryClient.setQueryData(habitKeys.list(), ctx.prev)
      }
    },
    onSettled: () => {
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
