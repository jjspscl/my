import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus } from 'lucide-react'
import { useCreateBill, useUpdateBill } from '../hooks/use-bills'
import type { RecurringBill, CreateBill } from '../schemas/bill.schemas'

interface BillFormDialogProps {
  trigger?: React.ReactNode
  bill?: RecurringBill
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function BillFormDialog({ trigger, bill, open, onOpenChange }: BillFormDialogProps) {
  const [name, setName] = useState(bill?.name ?? '')
  const [category, setCategory] = useState(bill?.category ?? '')
  const [amount, setAmount] = useState(bill ? String(bill.amountCents / 100) : '')
  const [frequency, setFrequency] = useState<string>(bill?.frequency ?? 'monthly')
  const [dayOfMonth, setDayOfMonth] = useState(bill ? String(bill.dayOfMonth) : '1')
  const [startDate, setStartDate] = useState(bill?.startDate ?? '')
  const [autoMatch, setAutoMatch] = useState(bill?.autoMatch ?? false)
  const [matchPattern, setMatchPattern] = useState(bill?.matchPattern ?? '')

  const createBill = useCreateBill()
  const updateBill = useUpdateBill()
  const isPending = createBill.isPending || updateBill.isPending

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const amountCents = Math.round(parseFloat(amount) * 100)
    if (!name || !category || !amountCents || !startDate || !dayOfMonth) return

    const data: CreateBill = {
      name,
      category,
      amountCents,
      frequency: frequency as 'monthly' | 'weekly' | 'yearly',
      dayOfMonth: parseInt(dayOfMonth),
      startDate,
      autoMatch,
      matchPattern: matchPattern || null,
    }

    if (bill) {
      updateBill.mutate({ id: bill.id, data }, { onSuccess: () => onOpenChange?.(false) })
    } else {
      createBill.mutate(data, { onSuccess: () => onOpenChange?.(false) })
    }
  }

  const resetForm = () => {
    if (!bill) {
      setName('')
      setCategory('')
      setAmount('')
      setFrequency('monthly')
      setDayOfMonth('1')
      setStartDate('')
      setAutoMatch(false)
      setMatchPattern('')
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { onOpenChange?.(o); if (!o) resetForm() }}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            Add Bill
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="text-sm">{bill ? 'Edit Bill' : 'Add Bill'}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name" className="text-xs">Name</Label>
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} className="text-sm" placeholder="Netflix" required />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="category" className="text-xs">Category</Label>
              <Input id="category" value={category} onChange={(e) => setCategory(e.target.value)} className="text-sm" placeholder="Subscription" required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="amount" className="text-xs">Amount (PHP)</Label>
              <Input id="amount" type="number" step="0.01" min="0.01" value={amount} onChange={(e) => setAmount(e.target.value)} className="text-sm" placeholder="499.00" required />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="frequency" className="text-xs">Frequency</Label>
              <Select value={frequency} onValueChange={setFrequency}>
                <SelectTrigger className="text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="monthly">Monthly</SelectItem>
                  <SelectItem value="weekly">Weekly</SelectItem>
                  <SelectItem value="yearly">Yearly</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="dayOfMonth" className="text-xs">Day of Month</Label>
              <Input id="dayOfMonth" type="number" min="1" max="31" value={dayOfMonth} onChange={(e) => setDayOfMonth(e.target.value)} className="text-sm" disabled={frequency === 'weekly'} required />
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="startDate" className="text-xs">Start Date</Label>
            <Input id="startDate" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="text-sm" required />
          </div>
          <div className="flex items-center gap-2">
            <Switch id="autoMatch" checked={autoMatch} onCheckedChange={setAutoMatch} />
            <Label htmlFor="autoMatch" className="text-xs">Auto-match from transactions</Label>
          </div>
          {autoMatch && (
            <div className="space-y-2">
              <Label htmlFor="matchPattern" className="text-xs">Match pattern (optional)</Label>
              <Input id="matchPattern" value={matchPattern} onChange={(e) => setMatchPattern(e.target.value)} className="text-sm" placeholder="e.g. NETFLIX" />
            </div>
          )}
          <Button type="submit" className="w-full text-sm" disabled={isPending}>
            {isPending ? 'Saving...' : bill ? 'Update Bill' : 'Add Bill'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}