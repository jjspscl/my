import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import { financeKeys } from '../api/finance.keys'
import { createImport, listImports, rollbackImport } from '../api/import.api'
import type { CreateImport } from '../schemas/import.schemas'

export function useImports() {
  return useQuery({
    queryKey: financeKeys.importList(),
    queryFn: listImports,
    staleTime: 1000 * 30,
  })
}

export function useCreateImport() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateImport) => createImport(data),
    onSuccess: (batch) => {
      queryClient.invalidateQueries({ queryKey: financeKeys.importList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.transactionList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
      if (batch.summary.replay) {
        toast.info('This statement was already imported — no duplicate rows were added.')
      }
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Import failed.')
    },
  })
}

export function useRollbackImport() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => rollbackImport(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.importList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.transactionList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
    },
  })
}
