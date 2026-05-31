import { useState } from 'react'
import { Trash2, Wallet as WalletIcon, Target, PiggyBank, Landmark } from 'lucide-react'
import { motion } from 'motion/react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { useGoals, useDeleteGoal } from '../hooks/use-goals'
import { useWallets } from '../hooks/use-wallets'
import { GoalFormDialog } from './goal-form-dialog'
import { GoalContributionDialog } from './goal-contribution-dialog'
import type { GoalSummary } from '../schemas/goal.schemas'
import type { Wallet } from '../schemas/wallet.schemas'

function formatCents(cents: number): string {
  return `₱${(cents / 100).toLocaleString('en-PH', { minimumFractionDigits: 0 })}`
}

function formatDate(dateStr: string | null | undefined): string {
  if (!dateStr) return 'No target date'
  const date = new Date(dateStr + 'T00:00:00')
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

function statusBadge(status: string) {
  switch (status) {
    case 'achieved':
      return <Badge variant="outline" className="text-green-600 border-green-600 text-xs">Achieved</Badge>
    case 'behind':
      return <Badge variant="outline" className="text-destructive border-destructive text-xs">Behind</Badge>
    case 'not_started':
      return <Badge variant="outline" className="text-muted-foreground text-xs">Not started</Badge>
    default:
      return <Badge variant="outline" className="text-xs">In progress</Badge>
  }
}

function GoalCard({ goal, wallets, onEdit, onDelete, onContribute }: {
  goal: GoalSummary
  wallets: Wallet[] | undefined
  onEdit: (g: GoalSummary) => void
  onDelete: (id: string) => void
  onContribute: (g: GoalSummary) => void
}) {
  const targetWallet = wallets?.find((w) => w.id === goal.targetWalletId)
  const hasWallet = !!(goal.targetWalletId && targetWallet)
  return (
    <motion.div
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      className="border rounded-md p-3 space-y-3"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-medium truncate">{goal.name}</span>
          {statusBadge(goal.status)}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => onContribute(goal)}>
            Add progress
          </Button>
          <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => onEdit(goal)}>
            Edit
          </Button>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => onDelete(goal.id)}>
            <Trash2 className="h-3 w-3 text-muted-foreground hover:text-destructive" />
          </Button>
        </div>
      </div>

      <Progress
        value={goal.progressPercent}
        className={`h-2 ${goal.status === 'achieved' ? '[&>div]:bg-green-600' : goal.status === 'behind' ? '[&>div]:bg-destructive' : ''}`}
      />

      <div className="grid grid-cols-3 gap-2 text-xs">
        <div>
          <p className="text-muted-foreground">Saved</p>
          <p className="font-medium tabular-nums">{formatCents(goal.currentAmountCents)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Target</p>
          <p className="font-medium tabular-nums">{formatCents(goal.targetAmountCents)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Remaining</p>
          <p className="font-medium tabular-nums">{formatCents(goal.remainingAmountCents)}</p>
        </div>
      </div>

      <div className="flex items-center gap-4 text-xs text-muted-foreground flex-wrap">
        <span className="flex items-center gap-1">
          <Target className="h-3 w-3" />
          {formatDate(goal.targetDate)}
        </span>
        {goal.requiredMonthlyCents != null && (
          <span className="flex items-center gap-1">
            <PiggyBank className="h-3 w-3" />
            {formatCents(goal.requiredMonthlyCents)}/mo needed
          </span>
        )}
        {hasWallet && (
          <span className="flex items-center gap-1">
            <Landmark className="h-3 w-3" />
            → {targetWallet!.name}
          </span>
        )}
      </div>
    </motion.div>
  )
}

export function GoalsPage() {
  const { data: goals, isLoading } = useGoals()
  const { data: wallets, isLoading: walletsLoading } = useWallets()
  const deleteGoal = useDeleteGoal()
  const [editingGoal, setEditingGoal] = useState<GoalSummary | null>(null)
  const [contributingGoal, setContributingGoal] = useState<GoalSummary | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [contributionOpen, setContributionOpen] = useState(false)

  const totalTarget = goals?.reduce((sum, g) => sum + g.targetAmountCents, 0) ?? 0
  const totalSaved = goals?.reduce((sum, g) => sum + g.currentAmountCents, 0) ?? 0
  const totalRemaining = totalTarget - totalSaved

  const handleEdit = (g: GoalSummary) => {
    setEditingGoal(g)
    setFormOpen(true)
  }

  const handleContribute = (g: GoalSummary) => {
    setContributingGoal(g)
    setContributionOpen(true)
  }

  const handleDelete = (id: string) => {
    if (confirm('Delete this goal? All contributions will be removed.')) {
      deleteGoal.mutate(id)
    }
  }

  return (
    <div className="space-y-6">
      {/* Summary cards */}
      <div className="grid grid-cols-3 gap-3">
        <Card>
          <CardContent className="pt-4 pb-3 text-center">
            <p className="text-xs text-muted-foreground">Total Target</p>
            <p className="text-lg font-bold">{formatCents(totalTarget)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4 pb-3 text-center">
            <p className="text-xs text-muted-foreground">Total Saved</p>
            <p className="text-lg font-bold text-green-600">{formatCents(totalSaved)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4 pb-3 text-center">
            <p className="text-xs text-muted-foreground">Total Remaining</p>
            <p className="text-lg font-bold">{formatCents(totalRemaining)}</p>
          </CardContent>
        </Card>
      </div>

      {/* Goals */}
      <Card>
        <CardHeader className="pb-2 flex flex-row items-center justify-between">
          <CardTitle className="text-sm flex items-center gap-2">
            <WalletIcon className="h-4 w-4" />
            Savings Goals
          </CardTitle>
          <GoalFormDialog
            open={formOpen && !editingGoal}
            onOpenChange={(o) => { setFormOpen(o); if (!o) setEditingGoal(null) }}
            wallets={wallets}
            walletsLoading={walletsLoading}
          />
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading...</p>
          ) : !goals || goals.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No savings goals yet. Create one to start tracking.
            </p>
          ) : (
            <div className="space-y-3">
              {goals.map((g) => (
                <GoalCard
                  key={g.id}
                  goal={g}
                  wallets={wallets}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                  onContribute={handleContribute}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit dialog */}
      {editingGoal && (
        <GoalFormDialog
          goal={editingGoal}
          open={formOpen && !!editingGoal}
          onOpenChange={(o) => { setFormOpen(o); if (!o) setEditingGoal(null) }}
          wallets={wallets}
          walletsLoading={walletsLoading}
        />
      )}

      {/* Contribution dialog */}
      {contributingGoal && (
        <GoalContributionDialog
          goal={contributingGoal}
          open={contributionOpen}
          onOpenChange={(o) => { setContributionOpen(o); if (!o) setContributingGoal(null) }}
        />
      )}
    </div>
  )
}