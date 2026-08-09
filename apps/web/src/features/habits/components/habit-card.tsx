import { Check, Loader2 } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import type { PaletteToken } from '@/shared/theme/palette'
import { Button } from '@/components/ui/button'
import type { Habit } from '../schemas/habit.schemas'
import { useToggleHabit } from '../hooks/use-habits'
import { todayLocal } from '../api/habits.api'

interface HabitCardProps {
  habit: Habit
}

export function HabitCard({ habit }: HabitCardProps) {
  const toggle = useToggleHabit()

  const handleToggle = () => {
    // Explicit set-state + frozen date: offline the call is queued, and the
    // replay must complete the intended day idempotently.
    toggle.mutate({
      habitId: habit.id,
      date: todayLocal(),
      completed: !habit.completedToday,
    })
  }

  const color = habit.color as PaletteToken

  return (
    <div className="flex items-center gap-3 rounded-md border px-3 py-2.5">
      {/* Color dot */}
      <div className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: `var(--palette-${color})` }} />

      {/* Name + streak */}
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium truncate">{habit.name}</p>
        <p className="text-xs text-muted-foreground tabular-nums">
          {habit.frequency === 'weekly'
            ? `${habit.targetPerWeek}x/week`
            : 'daily'}
          {habit.currentStreak != null && habit.currentStreak > 0 && (
            <> &middot; {habit.currentStreak} day streak</>
          )}
        </p>
      </div>

      {/* Toggle button */}
      <Button
        variant={habit.completedToday ? 'default' : 'outline'}
        size="icon"
        aria-label={`Toggle ${habit.name}`}
        aria-pressed={habit.completedToday}
        className={cn(
          'h-8 w-8 shrink-0',
          habit.completedToday && 'bg-foreground text-background hover:bg-foreground/90',
        )}
        onClick={handleToggle}
        disabled={toggle.isPending}
      >
        {toggle.isPending && toggle.variables?.habitId === habit.id ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <Check className={cn('h-4 w-4', habit.completedToday ? 'opacity-100' : 'opacity-0')} />
        )}
      </Button>
    </div>
  )
}