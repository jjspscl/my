import { useState } from 'react'
import { Loader2, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  useDeleteTransaction,
  useTransactions,
} from '@/features/finance/hooks/use-transactions'
import { formatCents } from '@/features/finance/lib/format'
import { toLocalDateStr, todayLocalStr } from '@/shared/lib/utils'

export function TransactionsPage() {
  const today = todayLocalStr()
  const thirtyDaysAgo = new Date()
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
  const defaultFrom = toLocalDateStr(thirtyDaysAgo)

  const [from, setFrom] = useState(defaultFrom)
  const [to, setTo] = useState(today)

  const { data, isLoading, isError } = useTransactions(from, to)
  const deleteTx = useDeleteTransaction()
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const handleDelete = (id: string) => {
    setDeletingId(id)
    deleteTx.mutate(id, {
      onSettled: () => setDeletingId(null),
    })
  }

  const transactions = data ?? []

  return (
    <div className="space-y-4">
      {/* Date filter */}
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-1.5">
          <label className="text-xs text-muted-foreground">From</label>
          <Input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="w-[150px] text-xs"
          />
        </div>
        <div className="flex items-center gap-1.5">
          <label className="text-xs text-muted-foreground">To</label>
          <Input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="w-[150px] text-xs"
          />
        </div>
      </div>

      {/* Transactions table */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <div className="flex flex-col items-center gap-2 py-12 text-center">
          <p className="text-sm text-muted-foreground">Failed to load transactions</p>
        </div>
      ) : transactions.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-12 text-center">
          <p className="text-sm text-muted-foreground">No transactions found</p>
          <p className="text-xs text-muted-foreground">Add your first transaction to get started</p>
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-xs">Date</TableHead>
                <TableHead className="text-xs">Category</TableHead>
                <TableHead className="text-xs">Description</TableHead>
                <TableHead className="text-xs text-right">Amount</TableHead>
                <TableHead className="w-10"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {transactions.map((tx) => (
                <TableRow key={tx.id}>
                  <TableCell className="text-xs tabular-nums text-muted-foreground">
                    {tx.transactionDate}
                  </TableCell>
                  <TableCell className="text-xs">{tx.category}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {tx.description || '-'}
                  </TableCell>
                  <TableCell
                    className={`text-xs tabular-nums text-right font-medium ${
                      tx.type === 'expense' ? 'text-red-600' : 'text-green-600'
                    }`}
                  >
                    {tx.type === 'expense' ? '-' : '+'}
                    {formatCents(tx.amountCents)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => handleDelete(tx.id)}
                      disabled={deletingId === tx.id}
                    >
                      {deletingId === tx.id ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <Trash2 className="h-3 w-3 text-muted-foreground hover:text-red-600" />
                      )}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}