import { createFileRoute } from '@tanstack/react-router'
import { AnalyticsOverviewPage } from '@/features/finance/components/analytics-overview-page'

export const Route = createFileRoute('/_authenticated/finance/analytics')({
  component: AnalyticsOverviewPage,
})