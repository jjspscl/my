import { z } from 'zod'

export const ClassificationSchema = z.enum([
  'needs',
  'wants',
  'savings',
  'income',
  'debt',
  'other',
  'unclassified',
])
export type Classification = z.infer<typeof ClassificationSchema>

export const CategorySchema = z.object({
  name: z.string(),
  classification: ClassificationSchema,
  essential: z.boolean(),
  active: z.boolean(),
})
export type Category = z.infer<typeof CategorySchema>

export const CategoryListSchema = z.object({
  data: z.array(CategorySchema),
})
export type CategoryList = z.infer<typeof CategoryListSchema>

export const UpdateCategorySchema = z.object({
  classification: ClassificationSchema,
  essential: z.boolean(),
  active: z.boolean(),
})
export type UpdateCategory = z.infer<typeof UpdateCategorySchema>