import { useAnomalies } from '@/features/finance/hooks/use-analytics'
import { formatCents } from '@/features/finance/lib/format'
import { SectionCard } from './section-card'

const DEFAULT_CURRENCY = 'PHP'

export function AnomaliesSection() {
  const { data, isLoading, isError, error } = useAnomalies(DEFAULT_CURRENCY)

  return (
    <SectionCard
      title="Spending Anomalies"
      loading={isLoading}
      error={isError ? error : null}
      isEmpty={!!data && !data.sufficient}
      emptyMessage="Not enough history to detect anomalies"
    >
      {data && data.anomalies.length === 0 ? (
        <p className="text-sm text-muted-foreground">No unusual spending detected</p>
      ) : (
        <div className="divide-y rounded-md border">
          {data?.anomalies.map((a) => (
            <div key={`${a.category}-${a.month}`} className="space-y-0.5 px-3 py-2">
              <div className="flex items-center justify-between text-xs">
                <span className="font-medium">{a.category}</span>
                <span className="tabular-nums">{formatCents(a.amountCents, a.currency)}</span>
              </div>
              <p className="text-xs text-muted-foreground">
                {a.month} · {a.explanation}
              </p>
            </div>
          ))}
        </div>
      )}
    </SectionCard>
  )
}