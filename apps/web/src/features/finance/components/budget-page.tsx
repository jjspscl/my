import { useState } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { motion, AnimatePresence } from 'motion/react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { useBudgetSummary } from '../hooks/use-budgets'
import { BudgetDonutChart } from './budget-donut-chart'
import { BudgetAllocationDialog } from './budget-allocation-dialog'
import type { BudgetCategorySummary } from '../schemas/budget.schemas'

function formatMonth(month: string): string {
  const parts = month.split('-')
  const year = parseInt(parts[0] || '0')
  const m = parseInt(parts[1] || '1') - 1
  const date = new Date(year, m)
  return date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
}

function getCurrentMonth(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

function shiftMonth(month: string, delta: number): string {
  const parts = month.split('-').map(Number)
  const year = parts[0] || 0
  const m = (parts[1] || 1) - 1 + delta
  const date = new Date(year, m)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

function formatCents(cents: number): string {
  return `₱${(cents / 100).toLocaleString('en-PH', { minimumFractionDigits: 0 })}`
}

function CategoryRow({ cat }: { cat: BudgetCategorySummary }) {
  const percent = cat.allocatedCents > 0
    ? Math.min(100, Math.round((cat.spentCents / cat.allocatedCents) * 100))
    : 0
  const overspent = cat.remainingCents < 0

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex items-center gap-3 py-2"
    >
      <div className="flex-1 min-w-0">
        <div className="flex justify-between text-sm mb-1">
          <span className="font-medium truncate">{cat.category}</span>
          <span className={overspent ? 'text-destructive font-semibold' : 'text-muted-foreground'}>
            {formatCents(cat.spentCents)} / {formatCents(cat.allocatedCents)}
          </span>
        </div>
        <Progress
          value={percent}
          className={`h-2 ${overspent ? '[&>div]:bg-destructive' : ''}`}
        />
      </div>
      <div className="text-xs text-muted-foreground w-16 text-right shrink-0">
        {overspent ? (
          <span className="text-destructive">-{formatCents(Math.abs(cat.remainingCents))}</span>
        ) : (
          formatCents(cat.remainingCents)
        )}
      </div>
    </motion.div>
  )
}

export function BudgetPage() {
  const [month, setMonth] = useState(getCurrentMonth)
  const { data: summary, isLoading } = useBudgetSummary(month)

  return (
    <div className="space-y-6">
      {/* Month Navigator */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" size="icon" onClick={() => setMonth((m) => shiftMonth(m, -1))}>
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <h2 className="text-lg font-semibold">{formatMonth(month)}</h2>
        <Button variant="ghost" size="icon" onClick={() => setMonth((m) => shiftMonth(m, 1))}>
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>

      {/* Summary Cards */}
      {summary && (
        <div className="grid grid-cols-3 gap-3">
          <Card>
            <CardContent className="pt-4 pb-3 text-center">
              <p className="text-xs text-muted-foreground">Allocated</p>
              <motion.p
                className="text-lg font-bold"
                key={summary.totalAllocatedCents}
                initial={{ scale: 0.8 }}
                animate={{ scale: 1 }}
              >
                {formatCents(summary.totalAllocatedCents)}
              </motion.p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4 pb-3 text-center">
              <p className="text-xs text-muted-foreground">Spent</p>
              <motion.p
                className="text-lg font-bold"
                key={summary.totalSpentCents}
                initial={{ scale: 0.8 }}
                animate={{ scale: 1 }}
              >
                {formatCents(summary.totalSpentCents)}
              </motion.p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-4 pb-3 text-center">
              <p className="text-xs text-muted-foreground">Remaining</p>
              <motion.p
                className={`text-lg font-bold ${summary.totalRemainingCents < 0 ? 'text-destructive' : 'text-green-600'}`}
                key={summary.totalRemainingCents}
                initial={{ scale: 0.8 }}
                animate={{ scale: 1 }}
              >
                {formatCents(summary.totalRemainingCents)}
              </motion.p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Donut Chart */}
      {summary && summary.categories.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Allocation Breakdown</CardTitle>
          </CardHeader>
          <CardContent>
            <BudgetDonutChart categories={summary.categories} />
          </CardContent>
        </Card>
      )}

      {/* Category List */}
      <Card>
        <CardHeader className="pb-2 flex flex-row items-center justify-between">
          <CardTitle className="text-sm">Categories</CardTitle>
          <BudgetAllocationDialog month={month} categories={summary?.categories ?? []} />
        </CardHeader>
        <CardContent>
          {isLoading && <p className="text-sm text-muted-foreground">Loading...</p>}
          {summary && summary.categories.length === 0 && (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No budget set for {formatMonth(month)}. Click Edit to allocate.
            </p>
          )}
          <AnimatePresence>
            {summary?.categories.map((cat) => (
              <CategoryRow key={cat.category} cat={cat} />
            ))}
          </AnimatePresence>
        </CardContent>
      </Card>
    </div>
  )
}