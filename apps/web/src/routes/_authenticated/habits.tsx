import { createFileRoute } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'

import { useHabits, useCompletionsMap, useArchiveHabit } from '@/features/habits/hooks/use-habits'
import { HabitCard } from '@/features/habits/components/habit-card'
import { AddHabitDialog } from '@/features/habits/components/add-habit-dialog'
import { HabitContributionGraph } from '@/features/habits/components/contribution-graph'

export const Route = createFileRoute('/_authenticated/habits')({
  component: HabitsPage,
})

function HabitsPage() {
  const { data: habits, isLoading, isError } = useHabits()
  const archive = useArchiveHabit()

  const today = new Date()
  const yearAgo = new Date(today)
  yearAgo.setFullYear(yearAgo.getFullYear() - 1)
  const fromStr = yearAgo.toISOString().split('T')[0]
  const toStr = today.toISOString().split('T')[0]
  const { data: completionsMap, isLoading: mapLoading } = useCompletionsMap(fromStr, toStr)

  if (isLoading) {
    return (
      <div className="p-4 flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-4">
        <p className="text-sm text-muted-foreground py-12 text-center">Failed to load habits</p>
      </div>
    )
  }

  const activeHabits = habits ?? []

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-medium">Habits</h1>
        <AddHabitDialog />
      </div>

      {/* Full year contribution graph */}
      <div className="rounded-md border p-3">
        <h2 className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">
          Past Year
        </h2>
        {mapLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : completionsMap ? (
          <div className="overflow-x-auto pb-1">
            <HabitContributionGraph data={completionsMap} weeks={52} color="--palette-green" />
          </div>
        ) : (
          <div className="text-center py-8 text-sm text-muted-foreground">
            No activity data yet
          </div>
        )}
      </div>

      {/* Active habits list */}
      <div className="space-y-1.5">
        <h2 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Habits
        </h2>
        {activeHabits.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            No habits yet. Create your first habit.
          </p>
        ) : (
          activeHabits.map((habit) => (
            <div key={habit.id} className="group flex items-center gap-1">
              <div className="flex-1">
                <HabitCard habit={habit} />
              </div>
              <button
                className="opacity-0 group-hover:opacity-100 text-xs text-muted-foreground hover:text-foreground px-1.5 py-1 transition-opacity"
                onClick={() => archive.mutate(habit.id)}
                title="Archive habit"
              >
                archive
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  )
}