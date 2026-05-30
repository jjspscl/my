import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts'
import type { BudgetCategorySummary } from '../schemas/budget.schemas'

const COLORS = [
  'hsl(var(--chart-1))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
  'hsl(220 14% 70%)',
  'hsl(150 40% 50%)',
  'hsl(30 80% 55%)',
  'hsl(280 40% 55%)',
]

interface Props {
  categories: BudgetCategorySummary[]
}

export function BudgetDonutChart({ categories }: Props) {
  const data = categories.map((c) => ({
    name: c.category,
    value: c.allocatedCents,
  }))

  if (data.length === 0) return null

  return (
    <ResponsiveContainer width="100%" height={200}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={50}
          outerRadius={80}
          paddingAngle={2}
          dataKey="value"
        >
          {data.map((_, idx) => (
            <Cell key={idx} fill={COLORS[idx % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip
          formatter={(value) => `₱${(Number(value) / 100).toLocaleString()}`}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}