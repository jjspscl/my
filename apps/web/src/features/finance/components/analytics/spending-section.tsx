import { useSpendingSummary } from '@/features/finance/hooks/use-analytics'
import { formatCents } from '@/features/finance/lib/format'
import { SectionCard } from './section-card'

export function SpendingSection() {
  const { data, isLoading, isError, error } = useSpendingSummary()

  return (
    <SectionCard title="Spending Breakdown" loading={isLoading} error={isError ? error : null}>
      {data?.currencies.map((c) => (
        <div key={c.currency} className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Total</span>
            <span className="font-medium tabular-nums">{formatCents(c.totalExpenseCents, c.currency)}</span>
          </div>
          <div className="divide-y rounded-md border">
            {Object.entries(c.byClassification).map(([classification, cents]) => (
              <div key={classification} className="flex items-center justify-between px-3 py-1.5 text-xs">
                <span className="capitalize text-muted-foreground">{classification}</span>
                <span className="tabular-nums">{formatCents(cents, c.currency)}</span>
              </div>
            ))}
            {c.unclassifiedCents > 0 && (
              <div className="flex items-center justify-between px-3 py-1.5 text-xs">
                <span className="text-muted-foreground">Unclassified</span>
                <span className="tabular-nums">{formatCents(c.unclassifiedCents, c.currency)}</span>
              </div>
            )}
          </div>
        </div>
      ))}
    </SectionCard>
  )
}