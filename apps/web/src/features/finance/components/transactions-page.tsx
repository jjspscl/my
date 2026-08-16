import { useState } from 'react'
import { Loader2, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'

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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  useDeleteTransaction,
  useTransactions,
} from '@/features/finance/hooks/use-transactions'
import { EditTransactionSheet } from './edit-transaction-sheet'
import { formatCents } from '@/features/finance/lib/format'
import { toLocalDateStr, todayLocalStr } from '@/shared/lib/utils'
import { useNetworkStatus } from '@/shared/sync/network-status'
import type { Transaction } from '../schemas/transaction.schemas'

export function TransactionsPage() {
  const today = todayLocalStr()
  const thirtyDaysAgo = new Date()
  thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30)
  const defaultFrom = toLocalDateStr(thirtyDaysAgo)

  const [from, setFrom] = useState(defaultFrom)
  const [to, setTo] = useState(today)

  const { data, isLoading, isError } = useTransactions(from, to)
  const deleteTx = useDeleteTransaction()
  const [deleting, setDeleting] = useState<Transaction | null>(null)
  const [editing, setEditing] = useState<Transaction | null>(null)
  const isOnline = useNetworkStatus((s) => s.isOnline)

  const transactions = data ?? []

  const handleDelete = () => {
    if (!deleting) return
    deleteTx.mutate(
      { id: deleting.id, revision: deleting.revision },
      {
        onSettled: () => setDeleting(null),
      },
    )
  }

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
                  <TableCell className="text-xs">
                    {tx.category}
                    {tx.imported && (
                      <span className="ml-1.5 text-[10px] uppercase tracking-wide text-muted-foreground">
                        imported
                      </span>
                    )}
                  </TableCell>
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
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          disabled={!isOnline}
                          aria-label={`Actions for ${tx.description || tx.category}, ${formatCents(tx.amountCents)}, ${tx.transactionDate}`}
                        >
                          {deleting?.id === tx.id && deleteTx.isPending ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <MoreHorizontal className="h-3.5 w-3.5" />
                          )}
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-40">
                        <DropdownMenuItem onClick={() => setEditing(tx)}>
                          <Pencil className="h-3.5 w-3.5" />
                          Edit transaction
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={() => setDeleting(tx)}
                          disabled={deleteTx.isPending}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                          Delete transaction
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {!isOnline && (
        <p className="text-xs text-muted-foreground">
          You are offline. Editing and deleting transactions is disabled until you reconnect.
        </p>
      )}

      {/* Edit sheet */}
      <EditTransactionSheet
        transaction={editing}
        open={editing !== null}
        onOpenChange={(open) => {
          if (!open) setEditing(null)
        }}
      />

      {/* Delete confirmation */}
      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this transaction?</AlertDialogTitle>
            <AlertDialogDescription>
              {deleting && (
                <span className="flex flex-col gap-2">
                  <span className="text-foreground">
                    {deleting.type === 'expense' ? '-' : '+'}
                    {formatCents(deleting.amountCents)} — {deleting.description || deleting.category}
                    {' '}on {deleting.transactionDate}
                  </span>
                  {deleting.imported ? (
                    <span>
                      This transaction came from an imported statement. Deleting it
                      removes the booked transaction; the original statement entry
                      stays, and this import is marked as modified.
                    </span>
                  ) : (
                    <span>This permanently removes the transaction. This cannot be undone.</span>
                  )}
                </span>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteTx.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault()
                handleDelete()
              }}
              disabled={deleteTx.isPending}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              {deleteTx.isPending ? 'Deleting...' : 'Delete transaction'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
