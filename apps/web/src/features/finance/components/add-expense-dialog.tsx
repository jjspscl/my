import { useState } from 'react'
import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { CategoryCombobox } from './category-combobox'
import { useCreateTransaction } from '../hooks/use-transactions'
import type { CreateTransaction } from '../schemas/transaction.schemas'

interface AddExpenseDialogProps {
  trigger?: React.ReactNode
  defaultType?: 'expense' | 'income'
}

export function AddExpenseDialog({ trigger, defaultType = 'expense' }: AddExpenseDialogProps) {
  const [open, setOpen] = useState(false)
  const [amountCents, setAmountCents] = useState('')
  const [category, setCategory] = useState('')
  const [description, setDescription] = useState('')
  const [type, setType] = useState<'expense' | 'income'>(defaultType)
  const [transactionDate, setTransactionDate] = useState(
    new Date().toISOString().split('T')[0],
  )

  const createTx = useCreateTransaction()

  const handleSubmit = () => {
    const cents = Math.round(parseFloat(amountCents) * 100)
    if (cents <= 0 || !category) return

    const data: CreateTransaction = {
      amountCents: cents,
      category,
      description,
      type,
      transactionDate,
    }

    createTx.mutate(data, {
      onSuccess: () => {
        setOpen(false)
        setAmountCents('')
        setCategory('')
        setDescription('')
        setType(defaultType)
        setTransactionDate(new Date().toISOString().split('T')[0])
      },
    })
  }

  const isValid = parseFloat(amountCents) > 0 && category

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm" className="w-full gap-2">
            <Plus className="h-4 w-4" />
            Add expense
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add {type === 'expense' ? 'Expense' : 'Income'}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="text-xs text-muted-foreground mb-1 block">Amount (PHP)</label>
              <Input
                type="number"
                step="0.01"
                min="0"
                placeholder="0.00"
                value={amountCents}
                onChange={(e) => setAmountCents(e.target.value)}
                autoFocus
              />
            </div>
            <div>
              <label className="text-xs text-muted-foreground mb-1 block">Type</label>
              <Select value={type} onValueChange={(v: 'expense' | 'income') => setType(v)}>
                <SelectTrigger className="w-[110px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="expense">Expense</SelectItem>
                  <SelectItem value="income">Income</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Category</label>
            <CategoryCombobox value={category} onChange={setCategory} />
          </div>

          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Description (optional)</label>
            <Input
              placeholder="Coffee, lunch..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Date</label>
            <Input
              type="date"
              value={transactionDate}
              onChange={(e) => setTransactionDate(e.target.value)}
            />
          </div>

          <Button
            className="w-full"
            size="sm"
            onClick={handleSubmit}
            disabled={!isValid || createTx.isPending}
          >
            {createTx.isPending ? 'Saving...' : `Add ${type === 'expense' ? 'Expense' : 'Income'}`}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}