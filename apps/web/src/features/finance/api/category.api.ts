import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import {
  CategoryListSchema,
  CategorySchema,
  type Category,
  type UpdateCategory,
} from '../schemas/category.schemas'

const CategoryDataSchema = z.object({
  ok: z.boolean().optional(),
  data: CategorySchema,
})

export async function listCategories(): Promise<Category[]> {
  const res = await apiClient('/api/v1/finance/categories', CategoryListSchema)
  return res.data
}

export async function updateCategory(name: string, data: UpdateCategory): Promise<Category> {
  const res = await apiClient(`/api/v1/finance/categories/${encodeURIComponent(name)}`, CategoryDataSchema, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
  return res.data
}