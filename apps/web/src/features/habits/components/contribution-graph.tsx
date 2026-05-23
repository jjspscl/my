import { useMemo, useState } from 'react'
import { cn } from '@/shared/lib/utils'
import type { CompletionsMap } from '../schemas/habit.schemas'

interface HabitContributionGraphProps {
  data: CompletionsMap
  weeks?: number        // 12 for widget, 52 for page
  color?: string        // CSS var name (e.g. '--palette-green')
}

interface Cell {
  date: string
  count: number
  total: number
  ratio: number  // 0–1
}

function buildDateRange(weeks: number): string[] {
  const days: string[] = []
  const now = new Date()
  // Align end to today (Saturday = 6, GitHub uses Sun=0)
  const totalDays = weeks * 7
  for (let i = totalDays - 1; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    days.push(d.toISOString()!.split('T')[0]!)
  }
  return days
}

function getMonthLabel(dates: string[], index: number): string | null {
  if (index >= dates.length) return null
  const d = new Date(dates[index] + 'T00:00:00')
  const prev = index > 0 ? new Date(dates[index - 1] + 'T00:00:00') : null
  if (!prev || d.getMonth() !== prev.getMonth()) {
    return d.toLocaleDateString('en-US', { month: 'short' })
  }
  return null
}

const DAY_LABELS = ['Mon', '', 'Wed', '', 'Fri', '', '']

export function HabitContributionGraph({ data, weeks = 12, color = '--palette-green' }: HabitContributionGraphProps) {
  const [tooltip, setTooltip] = useState<{ date: string; count: number; total: number; x: number; y: number } | null>(null)

  const days = useMemo(() => buildDateRange(weeks), [weeks])
  const totalHabits = data.totalHabits

  const cells: Cell[] = useMemo(() => {
    return days.map((date) => {
      const habitIds = data.completions[date]
      const count = habitIds ? habitIds.length : 0
      return {
        date,
        count,
        total: totalHabits,
        ratio: totalHabits > 0 ? count / totalHabits : 0,
      }
    })
  }, [days, data, totalHabits])

  // Group cells into weeks (7-day columns, rightmost = newest)
  const weeksArray: Cell[][] = []
  for (let w = 0; w < weeks; w++) {
    const week: Cell[] = []
    for (let d = 0; d < 7; d++) {
      const idx = w * 7 + d
      if (idx < cells.length) {
        week.push(cells[idx]!)
      }
    }
    weeksArray.push(week)
  }

  const getColorStyle = (ratio: number): React.CSSProperties => {
    const cssVar = `var(${color})`
    if (ratio <= 0.25) return { backgroundColor: cssVar, opacity: 0.25 }
    if (ratio <= 0.5) return { backgroundColor: cssVar, opacity: 0.5 }
    if (ratio <= 0.75) return { backgroundColor: cssVar, opacity: 0.75 }
    return { backgroundColor: cssVar }
  }

  const todayStr = new Date().toISOString().split('T')[0]

  return (
    <div className="relative">
      {/* Month labels */}
      <div className="flex ml-8 mb-0.5 text-[10px] text-muted-foreground">
        {weeksArray.map((week, wi) => {
          const firstDate = week[0]?.date
          if (!firstDate) return <div key={wi} className="w-[12px]" />
          const idx = days.indexOf(firstDate)
          const label = idx >= 0 ? getMonthLabel(days, idx) : null
          return (
            <div key={wi} className="w-[12px] text-[9px] leading-none" style={{ marginRight: '2px' }}>
              {label || ''}
            </div>
          )
        })}
      </div>

      <div className="flex gap-[2px]">
        {/* Day labels */}
        <div className="flex flex-col gap-[2px] mr-1 pt-0">
          {DAY_LABELS.map((label, i) => (
            <div key={i} className="h-[10px] text-[9px] leading-[10px] text-muted-foreground">
              {label}
            </div>
          ))}
        </div>

        {/* Grid */}
        <div className="flex gap-[2px]">
          {weeksArray.map((week, wi) => (
            <div key={wi} className="flex flex-col gap-[2px]">
              {week.map((cell, di) => (
                <div
                  key={`${wi}-${di}`}
                  className={cn(
                    'h-[10px] w-[10px] rounded-[2px] cursor-default transition-colors',
                    cell.ratio === 0 && 'bg-muted',
                    cell.date === todayStr && 'ring-1 ring-foreground/30',
                  )}
                  style={cell.ratio > 0 ? getColorStyle(cell.ratio) : undefined}
                  onMouseEnter={(e) => {
                    const rect = (e.target as HTMLElement).getBoundingClientRect()
                    setTooltip({
                      date: cell.date,
                      count: cell.count,
                      total: cell.total,
                      x: rect.left + rect.width / 2,
                      y: rect.top - 4,
                    })
                  }}
                  onMouseLeave={() => setTooltip(null)}
                />
              ))}
            </div>
          ))}
        </div>
      </div>

      {/* Tooltip */}
      {tooltip && (
        <div
          className="fixed z-50 px-2 py-1 rounded border bg-popover text-popover-foreground text-xs shadow-sm pointer-events-none whitespace-nowrap"
          style={{
            left: tooltip.x,
            top: tooltip.y,
            transform: 'translate(-50%, -100%)',
          }}
        >
          <span className="font-medium">{tooltip.date}</span>
          : {tooltip.count}/{tooltip.total} habits
        </div>
      )}

      {/* Legend */}
      <div className="flex items-center gap-1 mt-2 justify-end text-[10px] text-muted-foreground">
        <span>Less</span>
        <div className="h-[10px] w-[10px] rounded-[2px] bg-muted" />
        <div className="h-[10px] w-[10px] rounded-[2px]" style={{ backgroundColor: `var(${color})`, opacity: 0.25 }} />
        <div className="h-[10px] w-[10px] rounded-[2px]" style={{ backgroundColor: `var(${color})`, opacity: 0.5 }} />
        <div className="h-[10px] w-[10px] rounded-[2px]" style={{ backgroundColor: `var(${color})`, opacity: 0.75 }} />
        <div className="h-[10px] w-[10px] rounded-[2px]" style={{ backgroundColor: `var(${color})` }} />
        <span>More</span>
      </div>
    </div>
  )
}