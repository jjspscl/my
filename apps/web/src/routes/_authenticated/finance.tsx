import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { Plus, Trash2, Loader2 } from 'lucide-react'

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
import { AddExpenseDialog } from '@/features/finance/components/add-expense-dialog'
import {
  useTransactions,
  useDeleteTransaction,
} from '@/features/finance/hooks/use-transactions'

function formatPHP(cents: number): string {
  return new Intl.NumberFormat('en-PH', {
    style: 'currency',
    currency: 'PHP',
  }).format(cents / 100)
}

export const Route = createFileRoute('/_authenticated/finance')({
  component: FinancePage,
})

function FinancePage() {
  const today = new Date().toISOString().split('T')[0]
  const thirtyDaysAgo = new Date()
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
  const defaultFrom = thirtyDaysAgo.toISOString().split('T')[0]

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
                    {formatPHP(tx.amountCents)}
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