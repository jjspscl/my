import { createFileRoute } from '@tanstack/react-router'
import { BillsPage } from '@/features/finance/components/bills-page'

export const Route = createFileRoute('/_authenticated/finance/bills')({
  component: BillsPage,
})