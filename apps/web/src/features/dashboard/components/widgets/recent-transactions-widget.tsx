import { Receipt, Loader2 } from 'lucide-react'
import { useTransactions } from '@/features/finance/hooks/use-transactions'

function formatPHP(cents: number): string {
  return new Intl.NumberFormat('en-PH', {
    style: 'currency',
    currency: 'PHP',
  }).format(cents / 100)
}

export function RecentTransactionsWidget() {
  const today = new Date()
  const thirtyDaysAgo = new Date(today)
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)

  const from = thirtyDaysAgo.toISOString().split('T')[0]
  const to = today.toISOString().split('T')[0]

  const { data, isLoading, isError } = useTransactions(from, to)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex flex-col items-center gap-2 py-4 text-center">
        <Receipt className="h-6 w-6 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">Failed to load transactions</p>
      </div>
    )
  }

  const transactions = data ?? []

  if (transactions.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-4 text-center">
        <Receipt className="h-6 w-6 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">No transactions yet</p>
      </div>
    )
  }

  // Show last 5
  const recent = transactions.slice(0, 5)

  return (
    <div className="divide-y">
      {recent.map((tx) => (
        <div key={tx.id} className="flex items-center justify-between py-1.5 first:pt-0 last:pb-0">
          <div className="min-w-0 flex-1">
            <p className="text-xs font-medium truncate">
              {tx.description || tx.category}
            </p>
            <p className="text-[11px] text-muted-foreground">{tx.category}</p>
          </div>
          <span
            className={`ml-2 text-xs tabular-nums ${
              tx.type === 'expense' ? 'text-red-600' : 'text-green-600'
            }`}
          >
            {tx.type === 'expense' ? '-' : '+'}
            {formatPHP(tx.amountCents)}
          </span>
        </div>
      ))}
    </div>
  )
}