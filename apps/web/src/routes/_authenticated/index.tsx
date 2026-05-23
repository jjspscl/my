import { createFileRoute } from '@tanstack/react-router'
import { DashboardGrid } from '@/features/dashboard/components/dashboard-grid'

// Side-effect import registers widgets before first render
import '@/features/dashboard/lib/register-widgets'

export const Route = createFileRoute('/_authenticated/')({
  component: DashboardPage,
})

function DashboardPage() {
  return (
    <div className="p-4">
      <DashboardGrid />
    </div>
  )
}