import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import { listGoals, createGoal, updateGoal, deleteGoal, addContribution } from '../api/goal.api'
import type { CreateGoal } from '../schemas/goal.schemas'

export function useGoals() {
  return useQuery({
    queryKey: financeKeys.goalList(),
    queryFn: listGoals,
    staleTime: 1000 * 30,
  })
}

export function useCreateGoal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateGoal) => createGoal(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.goalList() })
    },
  })
}

export function useUpdateGoal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateGoal }) => updateGoal(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.goalList() })
    },
  })
}

export function useDeleteGoal() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteGoal(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.goalList() })
    },
  })
}

export function useAddContribution() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ goalId, amountCents, contributedAt, note, sourceWalletId, idempotencyKey }: { goalId: string; amountCents: number; contributedAt: string; note: string; sourceWalletId?: string; idempotencyKey?: string }) =>
      addContribution(goalId, amountCents, contributedAt, note, sourceWalletId, idempotencyKey),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.goalList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
    },
  })
}