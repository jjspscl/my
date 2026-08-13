import { useEmergencyFund } from '@/features/finance/hooks/use-analytics'
import { formatCents } from '@/features/finance/lib/format'
import { SectionCard } from './section-card'

const DEFAULT_CURRENCY = 'PHP'

export function EmergencyFundSection() {
  const { data, isLoading, isError, error } = useEmergencyFund(DEFAULT_CURRENCY)

  return (
    <SectionCard title="Emergency Fund" loading={isLoading} error={isError ? error : null}>
      {data ? (
        <div className="space-y-3">
          <div className="flex items-baseline justify-between">
            <span className="text-sm text-muted-foreground">Months of runway</span>
            <span className="text-lg font-medium tabular-nums">{data.monthsOfRunway.toFixed(1)}</span>
          </div>
          <div className="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
            <div>
              <p>Liquid balance</p>
              <p className="tabular-nums">{formatCents(data.liquidBalanceCents, data.currency)}</p>
            </div>
            <div>
              <p>Essential / month</p>
              <p className="tabular-nums">{formatCents(data.monthlyEssentialCents, data.currency)}</p>
            </div>
          </div>
          <p className="text-xs text-muted-foreground">
            Target range: {data.targetRangeMonths[0]}–{data.targetRangeMonths[1]} months
          </p>
        </div>
      ) : null}
    </SectionCard>
  )
}