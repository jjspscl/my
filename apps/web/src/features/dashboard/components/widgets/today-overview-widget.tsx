import { useTodayTotal } from '@/features/finance/hooks/use-transactions'

function formatPHP(cents: number): string {
  return new Intl.NumberFormat('en-PH', {
    style: 'currency',
    currency: 'PHP',
  }).format(cents / 100)
}

export function TodayOverviewWidget() {
  const now = new Date()
  const dateStr = now.toLocaleDateString('en-US', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  })

  const hour = now.getHours()
  let greeting = 'Good evening'
  if (hour < 12) greeting = 'Good morning'
  else if (hour < 17) greeting = 'Good afternoon'

  const { data: todayTotal, isLoading } = useTodayTotal()

  return (
    <div className="space-y-3">
      <div>
        <p className="text-xs text-muted-foreground tabular-nums">{dateStr}</p>
        <p className="text-lg font-medium">{greeting}</p>
      </div>

      <div className="grid grid-cols-2 gap-2 pt-1">
        <div className="rounded-md border p-2">
          <p className="text-[11px] text-muted-foreground">Today spent</p>
          <p className="text-sm font-medium tabular-nums text-red-600">
            {isLoading ? '...' : todayTotal ? formatPHP(todayTotal.expenseCents) : '₱0.00'}
          </p>
        </div>
        <div className="rounded-md border p-2">
          <p className="text-[11px] text-muted-foreground">Today earned</p>
          <p className="text-sm font-medium tabular-nums text-green-600">
            {isLoading ? '...' : todayTotal ? formatPHP(todayTotal.incomeCents) : '₱0.00'}
          </p>
        </div>
      </div>
    </div>
  )
}