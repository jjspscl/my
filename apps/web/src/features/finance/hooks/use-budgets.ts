import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import { getBudgetSummary, upsertBudget } from '../api/budget.api'
import type { UpsertBudget } from '../schemas/budget.schemas'

export function useBudgetSummary(month: string) {
  return useQuery({
    queryKey: financeKeys.budgetSummary(month),
    queryFn: () => getBudgetSummary(month),
    staleTime: 1000 * 60,
    enabled: !!month,
  })
}

export function useUpsertBudget() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: UpsertBudget) => upsertBudget(data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: financeKeys.budgetSummary(variables.month) })
    },
  })
}