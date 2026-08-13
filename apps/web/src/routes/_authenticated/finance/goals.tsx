import { createFileRoute } from '@tanstack/react-router'
import { GoalsPage } from '@/features/finance/components/goals-page'

export const Route = createFileRoute('/_authenticated/finance/goals')({
  component: GoalsPage,
})