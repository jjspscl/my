import { Loader2, BarChart3 } from 'lucide-react'
import { useCompletionsMap } from '@/features/habits/hooks/use-habits'
import { HabitContributionGraph } from '@/features/habits/components/contribution-graph'

export function HabitActivityWidget() {
  const today = new Date()
  const from = new Date(today)
  from.setDate(from.getDate() - 84) // 12 weeks
  const fromStr = from.toISOString().split('T')[0]
  const toStr = today.toISOString().split('T')[0]

  const { data, isLoading, isError } = useCompletionsMap(fromStr, toStr)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-6">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="flex flex-col items-center gap-2 py-6 text-center">
        <BarChart3 className="h-6 w-6 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">Failed to load activity</p>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto pb-1">
      <HabitContributionGraph data={data} weeks={12} color="--palette-green" />
    </div>
  )
}