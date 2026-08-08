import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useAddContribution } from '../hooks/use-goals'
import { useWallets } from '../hooks/use-wallets'
import type { GoalSummary } from '../schemas/goal.schemas'
import type { Wallet } from '../schemas/wallet.schemas'
import { todayLocalStr } from '@/shared/lib/utils'

interface GoalContributionDialogProps {
  goal: GoalSummary
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function GoalContributionDialog({ goal, open, onOpenChange }: GoalContributionDialogProps) {
  const [amount, setAmount] = useState('')
  const [contributedAt, setContributedAt] = useState(todayLocalStr())
  const [note, setNote] = useState('')
  const [sourceWalletId, setSourceWalletId] = useState('')

  const { data: wallets } = useWallets()
  const addContribution = useAddContribution()
  const isPending = addContribution.isPending

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const cents = Math.round(parseFloat(amount) * 100)
    if (!cents) return

    const swId = sourceWalletId && sourceWalletId.trim() ? sourceWalletId : undefined
    addContribution.mutate(
      { goalId: goal.id, amountCents: cents, contributedAt, note, sourceWalletId: swId },
      {
        onSuccess: () => {
          onOpenChange(false)
          setAmount('')
          setNote('')
          setSourceWalletId('')
        },
      },
    )
  }

  const remainingFormatted = (goal.remainingAmountCents / 100).toLocaleString('en-PH', {
    style: 'currency',
    currency: 'PHP',
    minimumFractionDigits: 0,
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle className="text-sm">Add Progress — {goal.name}</DialogTitle>
        </DialogHeader>

        <div className="text-xs text-muted-foreground mb-2">
          Remaining: <span className="font-medium tabular-nums">{remainingFormatted}</span>
          {' · '}Progress: {goal.progressPercent}%
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="contrib-amount" className="text-xs">Amount (PHP)</Label>
            <Input
              id="contrib-amount"
              type="number"
              step="0.01"
              min="0.01"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="text-sm"
              placeholder="1,000"
              autoFocus
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="contrib-date" className="text-xs">Date</Label>
            <Input id="contrib-date" type="date" value={contributedAt} onChange={(e) => setContributedAt(e.target.value)} className="text-sm" required />
          </div>
          {wallets && wallets.length > 0 && goal.targetWalletId && (
            <div className="space-y-2">
              <Label htmlFor="contrib-wallet" className="text-xs">Source Wallet (optional)</Label>
              <Select value={sourceWalletId} onValueChange={setSourceWalletId}>
                <SelectTrigger className="text-sm">
                  <SelectValue placeholder="Select source wallet" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value=" ">None</SelectItem>
                  {wallets.map((w: Wallet) => (
                    <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-[10px] text-muted-foreground">Auto-creates a transfer from source to goal target wallet</p>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="contrib-note" className="text-xs">Note (optional)</Label>
            <Input id="contrib-note" value={note} onChange={(e) => setNote(e.target.value)} className="text-sm" placeholder="First deposit" />
          </div>
          <Button type="submit" className="w-full text-sm" disabled={isPending}>
            {isPending ? 'Saving...' : 'Add Progress'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}