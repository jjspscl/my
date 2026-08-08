import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '../api/finance.keys'
import { listCategories, updateCategory } from '../api/category.api'
import type { UpdateCategory } from '../schemas/category.schemas'

export function useCategories() {
  return useQuery({
    queryKey: financeKeys.categoryList(),
    queryFn: listCategories,
    staleTime: 1000 * 60,
  })
}

export function useUpdateCategory() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ name, data }: { name: string; data: UpdateCategory }) =>
      updateCategory(name, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: financeKeys.categoryList() })
    },
  })
}