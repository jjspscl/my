import { createFileRoute } from '@tanstack/react-router'
import { TransactionsPage } from '@/features/finance/components/transactions-page'

export const Route = createFileRoute('/_authenticated/finance/')({
  component: TransactionsPage,
})