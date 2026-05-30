import { useState, useEffect } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus } from 'lucide-react'
import { useCreateGoal, useUpdateGoal } from '../hooks/use-goals'
import { useWallets } from '../hooks/use-wallets'
import type { GoalSummary } from '../schemas/goal.schemas'
import type { Wallet } from '../schemas/wallet.schemas'

interface GoalFormDialogProps {
  trigger?: React.ReactNode
  goal?: GoalSummary
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export function GoalFormDialog({ trigger, goal, open, onOpenChange }: GoalFormDialogProps) {
  const [name, setName] = useState(goal?.name ?? '')
  const [targetAmount, setTargetAmount] = useState(goal ? String(goal.targetAmountCents / 100) : '')
  const [targetDate, setTargetDate] = useState(goal?.targetDate ?? '')
  const [targetWalletId, setTargetWalletId] = useState(goal?.targetWalletId ?? '')

  const { data: wallets } = useWallets()
  const createGoal = useCreateGoal()
  const updateGoal = useUpdateGoal()
  const isPending = createGoal.isPending || updateGoal.isPending

  useEffect(() => {
    if (goal) {
      setName(goal.name)
      setTargetAmount(String(goal.targetAmountCents / 100))
      setTargetDate(goal.targetDate ?? '')
      setTargetWalletId(goal.targetWalletId ?? '')
    } else if (!open) {
      setName('')
      setTargetAmount('')
      setTargetDate('')
      setTargetWalletId('')
    }
  }, [goal, open])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const cents = Math.round(parseFloat(targetAmount) * 100)
    if (!name || !cents) return

    if (!targetWalletId) return;
    const data = {
      name,
      targetAmountCents: cents,
      targetDate: targetDate || null,
      targetWalletId,
    }

    if (goal) {
      updateGoal.mutate({ id: goal.id, data }, { onSuccess: () => onOpenChange?.(false) })
    } else {
      createGoal.mutate(data, { onSuccess: () => onOpenChange?.(false) })
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { onOpenChange?.(o) }}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            Add Goal
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="text-sm">{goal ? 'Edit Goal' : 'New Savings Goal'}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="goal-name" className="text-xs">Name</Label>
            <Input id="goal-name" value={name} onChange={(e) => setName(e.target.value)} className="text-sm" placeholder="Emergency Fund" required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="goal-target" className="text-xs">Target Amount (PHP)</Label>
            <Input id="goal-target" type="number" step="0.01" min="0.01" value={targetAmount} onChange={(e) => setTargetAmount(e.target.value)} className="text-sm" placeholder="500,000" required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="goal-date" className="text-xs">Target Date (optional)</Label>
            <Input id="goal-date" type="date" value={targetDate} onChange={(e) => setTargetDate(e.target.value)} className="text-sm" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="goal-wallet" className="text-xs">Target Wallet</Label>
            <Select value={targetWalletId} onValueChange={setTargetWalletId}>
              <SelectTrigger className="text-sm">
                <SelectValue placeholder="Select wallet" />
              </SelectTrigger>
              <SelectContent>
                {wallets?.map((w: Wallet) => (
                  <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button type="submit" className="w-full text-sm" disabled={isPending}>
            {isPending ? 'Saving...' : goal ? 'Update Goal' : 'Create Goal'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}