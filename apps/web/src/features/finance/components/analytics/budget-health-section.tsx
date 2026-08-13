import { useBudgetHealth } from '@/features/finance/hooks/use-analytics'
import { currentMonth, formatCents } from '@/features/finance/lib/format'
import { Progress } from '@/components/ui/progress'
import { SectionCard } from './section-card'

export function BudgetHealthSection() {
  const { data, isLoading, isError, error } = useBudgetHealth(currentMonth())

  return (
    <SectionCard title="Budget Health" loading={isLoading} error={isError ? error : null}>
      {data && !data.hasBudget ? (
        <p className="text-sm text-muted-foreground">No budget set for {data.month}</p>
      ) : data ? (
        <div className="space-y-3">
          <div className="grid grid-cols-3 gap-2 text-sm">
            <div>
              <p className="text-xs text-muted-foreground">Allocated</p>
              <p className="font-medium tabular-nums">{formatCents(data.totalAllocatedCents, data.currency)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Spent</p>
              <p className="font-medium tabular-nums">{formatCents(data.totalSpentCents, data.currency)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Remaining</p>
              <p className="font-medium tabular-nums">{formatCents(data.totalRemainingCents, data.currency)}</p>
            </div>
          </div>
          {data.unbudgetedSpentCents > 0 && (
            <p className="text-xs text-muted-foreground">
              Unbudgeted: {formatCents(data.unbudgetedSpentCents, data.currency)}
            </p>
          )}
          <div className="space-y-2">
            {data.categories.map((cat) => {
              const pct =
                cat.allocatedCents > 0
                  ? Math.min(100, Math.round((cat.spentCents / cat.allocatedCents) * 100))
                  : 0
              return (
                <div key={cat.category} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span>{cat.category}</span>
                    <span className="tabular-nums text-muted-foreground">
                      {formatCents(cat.spentCents, data.currency)} / {formatCents(cat.allocatedCents, data.currency)}
                    </span>
                  </div>
                  <Progress value={pct} className="h-1.5" />
                </div>
              )
            })}
          </div>
        </div>
      ) : null}
    </SectionCard>
  )
}