import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { financeKeys } from '../api/finance.keys'
import { listWallets, createWallet, updateWallet, archiveWallet } from '../api/wallet.api'
import type { CreateWallet } from '../schemas/wallet.schemas'

export function useWallets() {
  return useQuery({
    queryKey: financeKeys.walletList(),
    queryFn: listWallets,
    staleTime: 1000 * 30,
  })
}

export function useCreateWallet() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateWallet) => createWallet(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Could not create the wallet.')
    },
  })
}

export function useUpdateWallet() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateWallet }) => updateWallet(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
    },
  })
}

export function useArchiveWallet() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => archiveWallet(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.walletList() })
    },
  })
}
