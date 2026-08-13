import { useSavingsRate } from '@/features/finance/hooks/use-analytics'
import { formatCents } from '@/features/finance/lib/format'
import { SectionCard } from './section-card'

export function SavingsRateSection() {
  const { data, isLoading, isError, error } = useSavingsRate()

  return (
    <SectionCard title="Savings Rate" loading={isLoading} error={isError ? error : null}>
      {data?.length === 0 ? (
        <p className="text-sm text-muted-foreground">No income or expense this period</p>
      ) : (
        data?.map((rate) => (
          <div key={rate.currency} className="space-y-2">
            <div className="flex items-baseline justify-between">
              <span className="text-sm text-muted-foreground">{rate.currency}</span>
              <span className="text-lg font-medium tabular-nums">
                {rate.zeroIncome ? '—' : `${rate.ratePercent.toFixed(1)}%`}
              </span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs text-muted-foreground">
              <div>
                <p>Income</p>
                <p className="tabular-nums">{formatCents(rate.incomeCents, rate.currency)}</p>
              </div>
              <div>
                <p>Expense</p>
                <p className="tabular-nums">{formatCents(rate.expenseCents, rate.currency)}</p>
              </div>
              <div>
                <p>Net</p>
                <p className="tabular-nums">{formatCents(rate.netCents, rate.currency)}</p>
              </div>
            </div>
          </div>
        ))
      )}
    </SectionCard>
  )
}