import { createFileRoute } from '@tanstack/react-router'
import { WalletsPage } from '@/features/finance/components/wallets-page'

export const Route = createFileRoute('/_authenticated/finance/wallets')({
  component: WalletsPage,
})