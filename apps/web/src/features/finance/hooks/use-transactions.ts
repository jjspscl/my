import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { financeKeys } from '../api/finance.keys'
import {
  createTransaction as createTransactionApi,
  deleteTransaction as deleteTransactionApi,
  getTodayTotal as getTodayTotalApi,
  listTransactions as listTransactionsApi,
} from '../api/finance.api'
import type { CreateTransaction, Transaction } from '../schemas/transaction.schemas'

export function useTodayTotal() {
  return useQuery({
    queryKey: financeKeys.todayTotal(),
    queryFn: getTodayTotalApi,
    staleTime: 1000 * 60,
  })
}

export function useTransactions(from = '', to = '') {
  return useQuery({
    queryKey: financeKeys.transactionList({ from, to }),
    queryFn: () => listTransactionsApi(from, to),
    staleTime: 1000 * 60,
  })
}

export function useCreateTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateTransaction) => createTransactionApi(data),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: financeKeys.transactions() })
      const prevTotal = queryClient.getQueryData(financeKeys.todayTotal())
      return { prevTotal }
    },
    onError: (err, _vars, ctx) => {
      if (ctx?.prevTotal) {
        queryClient.setQueryData(financeKeys.todayTotal(), ctx.prevTotal)
      }
      toast.error(err instanceof Error ? err.message : 'Could not save the transaction.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteTransactionApi(id),
    onMutate: async (id) => {
      // Cancel in-flight fetches
      await queryClient.cancelQueries({ queryKey: financeKeys.transactions() })

      // Snapshot all transaction list queries for rollback
      const queries = queryClient.getQueriesData<Transaction[]>({ queryKey: financeKeys.transactions() })
      const snapshots = queries.map(([key, data]) => ({ key, data }))

      // Optimistically remove from all cached lists
      for (const { key } of snapshots) {
        queryClient.setQueryData<Transaction[]>(key, (old) =>
          old ? old.filter((tx) => tx.id !== id) : old,
        )
      }

      return { snapshots }
    },
    onError: (err, _vars, ctx) => {
      // Rollback all lists
      if (ctx?.snapshots) {
        for (const { key, data } of ctx.snapshots) {
          queryClient.setQueryData(key, data)
        }
      }
      toast.error(err instanceof Error ? err.message : 'Could not delete the transaction.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}
