import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import {
  createTransaction as createTransactionApi,
  deleteTransaction as deleteTransactionApi,
  getTodayTotal as getTodayTotalApi,
  listTransactions as listTransactionsApi,
} from '../api/finance.api'
import type { CreateTransaction } from '../schemas/transaction.schemas'

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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteTransactionApi(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}