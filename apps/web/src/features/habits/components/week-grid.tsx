import { cn } from '@/shared/lib/utils'
import type { PaletteToken } from '@/shared/theme/palette'
import type { Habit } from '../schemas/habit.schemas'

interface WeekGridProps {
  habits: Habit[]
}

function getLast7Days(): string[] {
  const days: string[] = []
  const now = new Date()
  for (let i = 6; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    days.push(d.toISOString()?.split('T')[0] ?? '')
  }
  return days
}

function getDayLabel(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString('en-US', { weekday: 'short' })
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00')
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

export function WeekGrid({ habits }: WeekGridProps) {
  const days = getLast7Days()

  if (habits.length === 0) {
    return (
      <div className="text-center py-8 text-sm text-muted-foreground">
        No habits yet. Add one to get started.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr>
            <th className="text-left font-normal text-muted-foreground py-1 pr-3 w-28" />
            {days.map((day) => (
              <th key={day} className="text-center font-normal text-muted-foreground py-1 px-1">
                <div>{getDayLabel(day)}</div>
                <div className="tabular-nums">{formatDate(day)}</div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {habits.map((habit) => {
            const color = habit.color as PaletteToken
            // For simplicity, we show today's completion status + past data
            // For past days we'd need the completions API — this is a placeholder
            // that shows today's status
            const todayIndex = days.length - 1

            return (
              <tr key={habit.id}>
                <td className="py-2 pr-3">
                  <div className="flex items-center gap-1.5">
                    <div className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: `var(--palette-${color})` }} />
                    <span className="truncate font-medium">{habit.name}</span>
                  </div>
                </td>
                {days.map((day, i) => {
                  // For now, only the "today" column shows actual status
                  // Future: fetch completions for the full week
                  const isToday = i === todayIndex
                  const filled = isToday && habit.completedToday
                  return (
                    <td key={day} className="text-center px-1 py-2">
                      <div
                        className={cn(
                          'mx-auto h-5 w-5 rounded-sm border',
                          filled
                            ? 'border-transparent'
                            : 'border-muted-foreground/20',
                        )}
                        style={filled ? { backgroundColor: `var(--palette-${color})` } : undefined}
                      />
                    </td>
                  )
                })}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}