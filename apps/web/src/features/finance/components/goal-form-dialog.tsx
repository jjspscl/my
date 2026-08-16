import { useState } from 'react'
import { motion } from 'motion/react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus } from 'lucide-react'
import { useCreateGoal, useUpdateGoal } from '../hooks/use-goals'
import { useMotionPreset } from '@/shared/lib/motion'
import type { GoalSummary } from '../schemas/goal.schemas'
import type { Wallet } from '../schemas/wallet.schemas'

interface GoalFormDialogProps {
  trigger?: React.ReactNode
  goal?: GoalSummary
  open?: boolean
  onOpenChange?: (open: boolean) => void
  wallets: Wallet[] | undefined
  walletsLoading?: boolean
}

export function GoalFormDialog({ trigger, goal, open, onOpenChange, wallets, walletsLoading }: GoalFormDialogProps) {
  const [name, setName] = useState(goal?.name ?? '')
  const [targetAmount, setTargetAmount] = useState(goal ? String(goal.targetAmountCents / 100) : '')
  const [targetDate, setTargetDate] = useState(goal?.targetDate ?? '')
  const [targetWalletId, setTargetWalletId] = useState(goal?.targetWalletId ?? '')

  const createGoal = useCreateGoal()
  const updateGoal = useUpdateGoal()
  const isPending = createGoal.isPending || updateGoal.isPending
  const preset = useMotionPreset()

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
        <motion.form
          variants={preset.container}
          initial="initial"
          animate="animate"
          onSubmit={handleSubmit}
          className="space-y-4"
        >
          <motion.div variants={preset.field} className="space-y-2">
            <Label htmlFor="goal-name" className="text-xs">Name</Label>
            <Input id="goal-name" value={name} onChange={(e) => setName(e.target.value)} className="text-sm" placeholder="Emergency Fund" required />
          </motion.div>
          <motion.div variants={preset.field} className="space-y-2">
            <Label htmlFor="goal-target" className="text-xs">Target Amount (PHP)</Label>
            <Input id="goal-target" type="number" step="0.01" min="0.01" value={targetAmount} onChange={(e) => setTargetAmount(e.target.value)} className="text-sm" placeholder="500,000" required />
          </motion.div>
          <motion.div variants={preset.field} className="space-y-2">
            <Label htmlFor="goal-date" className="text-xs">Target Date (optional)</Label>
            <Input id="goal-date" type="date" value={targetDate} onChange={(e) => setTargetDate(e.target.value)} className="text-sm" />
          </motion.div>
          <motion.div variants={preset.field} className="space-y-2">
            <Label htmlFor="goal-wallet" className="text-xs">Target Wallet</Label>
            {walletsLoading ? (
              <div className="flex h-9 items-center rounded-md border border-input px-3 text-sm text-muted-foreground">
                Loading wallets...
              </div>
            ) : !wallets || wallets.length === 0 ? (
              <div className="flex h-9 items-center rounded-md border border-input px-3 text-sm text-muted-foreground">
                No wallets — create one first
              </div>
            ) : (
              <Select value={targetWalletId || ''} onValueChange={setTargetWalletId}>
                <SelectTrigger id="goal-wallet" className="w-full text-sm">
                  <SelectValue placeholder="Select wallet" />
                </SelectTrigger>
                <SelectContent>
                  {wallets.map((w: Wallet) => (
                    <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </motion.div>
          <motion.div variants={preset.field}>
            <Button type="submit" className="w-full text-sm" disabled={isPending}>
              {isPending ? 'Saving...' : goal ? 'Update Goal' : 'Create Goal'}
            </Button>
          </motion.div>
        </motion.form>
      </DialogContent>
    </Dialog>
  )
}