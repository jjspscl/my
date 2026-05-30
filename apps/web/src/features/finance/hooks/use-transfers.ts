import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import { listTransfers, createTransfer } from '../api/transfer.api'
import type { CreateTransfer } from '../schemas/transfer.schemas'

export function useTransfers() {
  return useQuery({
    queryKey: financeKeys.transferList(),
    queryFn: listTransfers,
    staleTime: 1000 * 30,
  })
}

export function useCreateTransfer() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateTransfer) => createTransfer(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transferList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
    },
  })
}
