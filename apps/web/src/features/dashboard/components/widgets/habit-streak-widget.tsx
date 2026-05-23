import { CheckSquare, Loader2 } from 'lucide-react'
import { useHabits } from '@/features/habits/hooks/use-habits'
import { paletteBgClass, type PaletteToken } from '@/shared/theme/palette'

export function HabitStreakWidget() {
  const { data: habits, isLoading } = useHabits()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const activeHabits = habits ?? []
  const totalCount = activeHabits.length

  // Sort by streak descending, take top 3
  const topHabits = [...activeHabits]
    .sort((a, b) => (b.currentStreak ?? 0) - (a.currentStreak ?? 0))
    .slice(0, 3)

  if (totalCount === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-4 text-center">
        <CheckSquare className="h-6 w-6 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">No habits tracked</p>
      </div>
    )
  }

  return (
    <div className="space-y-2.5">
      {topHabits.map((habit) => {
        const color = habit.color as PaletteToken
        return (
          <div key={habit.id} className="flex items-center gap-2.5">
            <div className={`h-2.5 w-2.5 rounded-full shrink-0 ${paletteBgClass(color)}`} />
            <span className="text-xs flex-1 truncate">{habit.name}</span>
            {habit.currentStreak != null && habit.currentStreak > 0 && (
              <span className="text-xs tabular-nums text-muted-foreground">
                🔥{habit.currentStreak}
              </span>
            )}
          </div>
        )
      })}

      {totalCount > 3 && (
        <p className="text-[11px] text-muted-foreground">
          +{totalCount - 3} more habit{totalCount - 3 !== 1 ? 's' : ''}
        </p>
      )}
    </div>
  )
}