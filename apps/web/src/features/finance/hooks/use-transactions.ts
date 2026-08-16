import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { financeKeys } from '../api/finance.keys'
import {
  bulkDeleteTransactions as bulkDeleteTransactionsApi,
  bulkUpdateTransactions as bulkUpdateTransactionsApi,
  createTransaction as createTransactionApi,
  deleteTransaction as deleteTransactionApi,
  getTodayTotal as getTodayTotalApi,
  listTransactions as listTransactionsApi,
  updateTransaction as updateTransactionApi,
} from '../api/finance.api'
import type {
  BulkDeleteRequest,
  BulkUpdatePatch,
  BulkUpdateRequest,
  CreateTransaction,
  Transaction,
  UpdateTransaction,
} from '../schemas/transaction.schemas'
import { ApiError } from '@/shared/api/client'

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
    mutationFn: ({ id, revision }: { id: string; revision?: number }) =>
      deleteTransactionApi(id, revision),
    onMutate: async ({ id }) => {
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
      if (err instanceof ApiError && err.status === 412) {
        // The row moved on (edited elsewhere). Refetch so the UI shows truth.
        queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
        toast.error('Transaction changed elsewhere. Reloaded the latest list — try again.')
        return
      }
      toast.error(err instanceof Error ? err.message : 'Could not delete the transaction.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}

// useUpdateTransaction patches a transaction online. Optimistic: the cached
// lists and today total are patched, snapshotted for rollback. A 412 (stale
// revision) refetches truth and reports the conflict instead of overwriting.
export function useUpdateTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      data,
      revision,
    }: {
      id: string
      data: UpdateTransaction
      revision: number
    }) => updateTransactionApi(id, data, revision),
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: financeKeys.transactions() })
      const queries = queryClient.getQueriesData<Transaction[]>({ queryKey: financeKeys.transactions() })
      const snapshots = queries.map(([key, cached]) => ({ key, data: cached }))
      const prevTotal = queryClient.getQueryData(financeKeys.todayTotal())

      for (const { key } of snapshots) {
        queryClient.setQueryData<Transaction[]>(key, (old) =>
          old ? old.map((tx) => (tx.id === id ? { ...tx, ...data } : tx)) : old,
        )
      }

      return { snapshots, prevTotal }
    },
    onError: (err, _vars, ctx) => {
      if (ctx?.snapshots) {
        for (const { key, data } of ctx.snapshots) {
          queryClient.setQueryData(key, data)
        }
      }
      if (ctx?.prevTotal) {
        queryClient.setQueryData(financeKeys.todayTotal(), ctx.prevTotal)
      }
      if (err instanceof ApiError && err.status === 412) {
        queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
        toast.error('Transaction changed elsewhere. Reloaded the latest values — review and try again.')
        return
      }
      toast.error(err instanceof Error ? err.message : 'Could not save the changes.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.transactions() })
    },
  })
}

// useBulkUpdateTransactions applies one patch to many transactions. No
// optimistic update: partial failure would be too easy to misrepresent, so
// the cache is invalidated on settle and the result read from the server.
export function useBulkUpdateTransactions() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: BulkUpdateRequest) => bulkUpdateTransactionsApi(data),
    onSuccess: (_data, vars) => {
      toast.success(`Updated ${vars.items.length} transaction${vars.items.length === 1 ? '' : 's'}.`)
      queryClient.invalidateQueries({ queryKey: financeKeys.all })
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 412) {
        toast.error('Some transactions changed elsewhere. Reloaded the latest list — review and try again.')
        return
      }
      toast.error(err instanceof Error ? err.message : 'Could not update the transactions.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.all })
    },
  })
}

// useBulkDeleteTransactions removes many transactions atomically.
export function useBulkDeleteTransactions() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: BulkDeleteRequest) => bulkDeleteTransactionsApi(data),
    onSuccess: (_deleted, vars) => {
      toast.success(`Deleted ${vars.items.length} transaction${vars.items.length === 1 ? '' : 's'}.`)
      queryClient.invalidateQueries({ queryKey: financeKeys.all })
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 412) {
        toast.error('Some transactions changed elsewhere. Reloaded the latest list — review and try again.')
        return
      }
      toast.error(err instanceof Error ? err.message : 'Could not delete the transactions.')
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.all })
    },
  })
}

// bulkItem converts a Transaction into a bulk request item (id + revision).
export function bulkItem(tx: Transaction): BulkUpdateRequest['items'][number] {
  return { id: tx.id, revision: tx.revision }
}

// buildBulkPatch builds the request for the selected items from the toggled
// patch fields; untouched fields are omitted so the server leaves them alone.
export function buildBulkRequest(
  selected: Transaction[],
  patch: BulkUpdatePatch,
): BulkUpdateRequest {
  return {
    items: selected.map(bulkItem),
    patch,
  }
}
