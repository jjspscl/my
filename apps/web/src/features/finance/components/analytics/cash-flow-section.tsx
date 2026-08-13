import { useCashFlowSummary } from '@/features/finance/hooks/use-analytics'
import { formatCents } from '@/features/finance/lib/format'
import { SectionCard } from './section-card'

export function CashFlowSection() {
  const { data, isLoading, isError, error } = useCashFlowSummary()

  return (
    <SectionCard title="Cash Flow" loading={isLoading} error={isError ? error : null}>
      {data?.currencies.map((c) => (
        <div key={c.currency} className="space-y-3">
          <div className="grid grid-cols-3 gap-2 text-sm">
            <div>
              <p className="text-xs text-muted-foreground">Income</p>
              <p className="font-medium tabular-nums">{formatCents(c.incomeCents, c.currency)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Expense</p>
              <p className="font-medium tabular-nums">{formatCents(c.expenseCents, c.currency)}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Net</p>
              <p className="font-medium tabular-nums">{formatCents(c.netCents, c.currency)}</p>
            </div>
          </div>
          {c.monthly.length > 0 && (
            <div className="divide-y rounded-md border">
              {c.monthly.map((m) => (
                <div key={m.month} className="flex items-center justify-between px-3 py-1.5 text-xs">
                  <span className="text-muted-foreground">{m.month}</span>
                  <span className="tabular-nums">{formatCents(m.netCents, m.currency)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      ))}
    </SectionCard>
  )
}