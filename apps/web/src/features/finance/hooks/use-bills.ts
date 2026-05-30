import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import { listBills, createBill, updateBill, deleteBill, getUpcomingBills, markBillPaid } from '../api/bill.api'
import type { CreateBill } from '../schemas/bill.schemas'

export function useBills() {
  return useQuery({
    queryKey: financeKeys.billList(),
    queryFn: listBills,
    staleTime: 1000 * 60,
  })
}

export function useUpcomingBills(days = 30) {
  return useQuery({
    queryKey: financeKeys.upcomingBills(days),
    queryFn: () => getUpcomingBills(days),
    staleTime: 1000 * 30,
  })
}

export function useCreateBill() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateBill) => createBill(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.billList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.upcomingBills() })
    },
  })
}

export function useUpdateBill() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateBill }) => updateBill(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.billList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.upcomingBills() })
    },
  })
}

export function useDeleteBill() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => deleteBill(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.billList() })
      queryClient.invalidateQueries({ queryKey: financeKeys.upcomingBills() })
    },
  })
}

export function useMarkBillPaid() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ billId, dueDate, transactionId }: { billId: string; dueDate: string; transactionId?: string }) =>
      markBillPaid(billId, dueDate, transactionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.upcomingBills() })
      queryClient.invalidateQueries({ queryKey: financeKeys.billList() })
    },
  })
}