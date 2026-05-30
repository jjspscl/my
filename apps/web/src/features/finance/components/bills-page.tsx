import { useState } from 'react'
import { Trash2, Loader2, CircleCheck, CircleAlert, Clock } from 'lucide-react'
import { motion } from 'motion/react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useBills, useUpcomingBills, useDeleteBill, useMarkBillPaid } from '../hooks/use-bills'
import { BillFormDialog } from './bill-form-dialog'
import type { RecurringBill, UpcomingBill } from '../schemas/bill.schemas'

function formatCents(cents: number): string {
  return `₱${(cents / 100).toLocaleString('en-PH', { minimumFractionDigits: 2 })}`
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00')
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function statusIcon(status: string) {
  switch (status) {
    case 'paid':
      return <CircleCheck className="h-4 w-4 text-green-600" />
    case 'overdue':
      return <CircleAlert className="h-4 w-4 text-destructive" />
    default:
      return <Clock className="h-4 w-4 text-muted-foreground" />
  }
}

function statusBadge(status: string) {
  switch (status) {
    case 'paid':
      return <Badge variant="outline" className="text-green-600 border-green-600 text-xs">Paid</Badge>
    case 'overdue':
      return <Badge variant="outline" className="text-destructive border-destructive text-xs">Overdue</Badge>
    default:
      return <Badge variant="outline" className="text-xs">Upcoming</Badge>
  }
}

interface UpcomingRowProps {
  bill: UpcomingBill
  onMarkPaid: (id: string, dueDate: string) => void
  isPaying: boolean
}

function UpcomingRow({ bill, onMarkPaid, isPaying }: UpcomingRowProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex items-center gap-3 py-2 border-b border-border last:border-0"
    >
      <div className="shrink-0">{statusIcon(bill.status)}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate">{bill.name}</span>
          {statusBadge(bill.status)}
        </div>
        <p className="text-xs text-muted-foreground">
          Due {formatDate(bill.dueDate)} &middot; {formatCents(bill.amountCents)}
        </p>
      </div>
      <div className="shrink-0 text-right">
        <p className="text-sm font-medium tabular-nums">{formatCents(bill.amountCents)}</p>
        {bill.status !== 'paid' && (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 text-xs"
            onClick={() => onMarkPaid(bill.id, bill.dueDate)}
            disabled={isPaying}
          >
            {isPaying ? <Loader2 className="h-3 w-3 animate-spin" /> : 'Mark Paid'}
          </Button>
        )}
      </div>
    </motion.div>
  )
}

export function BillsPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingBill, setEditingBill] = useState<RecurringBill | null>(null)
  const [payingId, setPayingId] = useState<string | null>(null)

  const { data: bills, isLoading: billsLoading } = useBills()
  const { data: upcoming, isLoading: upcomingLoading } = useUpcomingBills(60)
  const deleteBill = useDeleteBill()
  const markPaid = useMarkBillPaid()

  const handleEdit = (bill: RecurringBill) => {
    setEditingBill(bill)
    setDialogOpen(true)
  }

  const handleDelete = (id: string) => {
    if (confirm('Delete this bill?')) {
      deleteBill.mutate(id)
    }
  }

  const handleMarkPaid = (billId: string, dueDate: string) => {
    setPayingId(`${billId}:${dueDate}`)
    markPaid.mutate({ billId, dueDate }, {
      onSettled: () => setPayingId(null),
    })
  }

  const handleDialogClose = (open: boolean) => {
    setDialogOpen(open)
    if (!open) {
      setEditingBill(null)
    }
  }

  return (
    <div className="space-y-6">
      {/* Upcoming Bills */}
      <Card>
        <CardHeader className="pb-2 flex flex-row items-center justify-between">
          <CardTitle className="text-sm">Upcoming Bills</CardTitle>
        </CardHeader>
        <CardContent>
          {upcomingLoading ? (
            <p className="text-sm text-muted-foreground">Loading...</p>
          ) : !upcoming || upcoming.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No upcoming bills. Add one to get started.
            </p>
          ) : (
            <div className="divide-y divide-border">
              {upcoming.map((b) => (
                <UpcomingRow
                  key={`${b.id}:${b.dueDate}`}
                  bill={b}
                  onMarkPaid={handleMarkPaid}
                  isPaying={payingId === `${b.id}:${b.dueDate}`}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* All Bills */}
      <Card>
        <CardHeader className="pb-2 flex flex-row items-center justify-between">
          <CardTitle className="text-sm">All Bills</CardTitle>
          <BillFormDialog
            open={dialogOpen}
            onOpenChange={handleDialogClose}
            bill={editingBill ?? undefined}
          />
        </CardHeader>
        <CardContent>
          {billsLoading ? (
            <p className="text-sm text-muted-foreground">Loading...</p>
          ) : !bills || bills.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No recurring bills yet.
            </p>
          ) : (
            <div className="divide-y divide-border">
              {bills.map((bill) => (
                <motion.div
                  key={bill.id}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="flex items-center gap-3 py-2"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{bill.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {bill.category} &middot; {bill.frequency} &middot; Day {bill.dayOfMonth}
                      {bill.autoMatch && ' \u00b7 Auto-match'}
                    </p>
                  </div>
                  <p className="text-sm font-medium tabular-nums">{formatCents(bill.amountCents)}</p>
                  <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => handleEdit(bill)}>
                    Edit
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => handleDelete(bill.id)}
                    disabled={deleteBill.isPending}
                  >
                    <Trash2 className="h-3 w-3 text-muted-foreground hover:text-destructive" />
                  </Button>
                </motion.div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}