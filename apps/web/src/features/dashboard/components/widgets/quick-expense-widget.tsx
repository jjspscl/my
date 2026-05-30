import { useState, useRef, useEffect } from 'react'
import { Plus, Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { CategoryCombobox } from '@/features/finance/components/category-combobox'
import { AddExpenseDialog } from '@/features/finance/components/add-expense-dialog'
import { useTodayTotal, useCreateTransaction } from '@/features/finance/hooks/use-transactions'

function formatPHP(cents: number): string {
  return new Intl.NumberFormat('en-PH', {
    style: 'currency',
    currency: 'PHP',
  }).format(cents / 100)
}

export function QuickExpenseWidget() {
  const [amount, setAmount] = useState('')
  const [category, setCategory] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const { data: todayTotal, isLoading: totalLoading } = useTodayTotal()
  const createTx = useCreateTransaction()

  // Focus input when mutation completes
  useEffect(() => {
    if (!createTx.isPending && !createTx.isSuccess) return
    if (createTx.isSuccess) {
      setAmount('')
      setCategory('')
      inputRef.current?.focus()
    }
  }, [createTx.isPending, createTx.isSuccess])

  const handleQuickAdd = () => {
    const cents = Math.round(parseFloat(amount) * 100)
    if (cents <= 0 || !category) return

    createTx.mutate({
      amountCents: cents,
      category,
      description: '',
      type: 'expense',
      walletId: '',
    })
  }

  const canSubmit = parseFloat(amount) > 0 && !!category && !createTx.isPending

  return (
    <div className="space-y-3">
      {/* Inline quick-add row */}
      <div className="flex gap-1.5">
        <Input
          ref={inputRef}
          type="number"
          step="0.01"
          min="0"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          className="w-[90px] text-sm tabular-nums"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && canSubmit) handleQuickAdd()
          }}
        />
        <div className="flex-1 min-w-0">
          <CategoryCombobox
            value={category}
            onChange={setCategory}
            placeholder="Category"
          />
        </div>
        <Button
          variant="default"
          size="icon"
          className="h-9 w-9 shrink-0"
          onClick={handleQuickAdd}
          disabled={!canSubmit}
        >
          {createTx.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Plus className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Today's total */}
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>Today&apos;s total</span>
        <span className="tabular-nums font-medium">
          {totalLoading
            ? '...'
            : todayTotal
              ? formatPHP(todayTotal.totalCents)
              : '₱0.00'}
        </span>
      </div>

      {/* Full dialog */}
      <AddExpenseDialog />
    </div>
  )
}