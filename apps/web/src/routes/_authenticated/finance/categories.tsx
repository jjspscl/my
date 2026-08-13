import { createFileRoute } from '@tanstack/react-router'
import { CategoriesPage } from '@/features/finance/components/categories-page'

export const Route = createFileRoute('/_authenticated/finance/categories')({
  component: CategoriesPage,
})