import { createFileRoute } from '@tanstack/react-router'
import { BudgetPage } from '@/features/finance/components/budget-page'

export const Route = createFileRoute('/_authenticated/finance/budget')({
  component: BudgetPage,
})