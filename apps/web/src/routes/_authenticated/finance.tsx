import { createFileRoute, Link, Outlet, useLocation } from '@tanstack/react-router'
import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { AddExpenseDialog } from '@/features/finance/components/add-expense-dialog'
import { cn } from '@/shared/lib/utils'

export const Route = createFileRoute('/_authenticated/finance')({
  component: FinanceLayout,
  pendingComponent: FinanceSkeleton,
  errorComponent: FinanceError,
})

const tabs = [
  { to: '/finance', label: 'Transactions' },
  { to: '/finance/budget', label: 'Budget' },
  { to: '/finance/bills', label: 'Bills' },
  { to: '/finance/goals', label: 'Goals' },
  { to: '/finance/wallets', label: 'Wallets' },
  { to: '/finance/categories', label: 'Categories' },
]

function FinanceLayout() {
  const location = useLocation()

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-medium">Finance</h1>
        <AddExpenseDialog
          trigger={
            <Button size="sm" className="gap-2">
              <Plus className="h-4 w-4" />
              Add Transaction
            </Button>
          }
        />
      </div>

      <nav className="flex flex-wrap items-center gap-1 rounded-lg bg-muted p-[3px]">
        {tabs.map((tab) => {
          const active = location.pathname === tab.to
          return (
            <Link
              key={tab.to}
              to={tab.to}
              className={cn(
                'rounded-md border border-transparent px-3 py-1.5 text-sm font-medium whitespace-nowrap transition-all',
                active
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-foreground/60 hover:text-foreground',
              )}
            >
              {tab.label}
            </Link>
          )
        })}
      </nav>

      <div className="pt-4">
        <Outlet />
      </div>
    </div>
  )
}

function FinanceSkeleton() {
  return (
    <div className="p-4 space-y-4">
      <Skeleton className="h-8 w-48" />
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <Skeleton className="h-64" />
    </div>
  )
}

function FinanceError({ error }: { error: Error }) {
  return (
    <div className="p-4">
      <div className="rounded-lg border border-destructive/50 p-4 text-center">
        <p className="text-sm text-destructive">{error.message}</p>
      </div>
    </div>
  )
}