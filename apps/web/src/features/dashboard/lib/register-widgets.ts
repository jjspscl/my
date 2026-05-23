import { registerWidget } from './widget-registry'
import { TodayOverviewWidget } from '../components/widgets/today-overview-widget'
import { QuickExpenseWidget } from '../components/widgets/quick-expense-widget'
import { RecentTransactionsWidget } from '../components/widgets/recent-transactions-widget'
import { HabitStreakWidget } from '../components/widgets/habit-streak-widget'
import { HabitActivityWidget } from '../components/widgets/habit-activity-widget'

// Register widgets at module level — runs on import, idempotent (skip if exists)
registerWidget(
  { id: 'today-overview', title: 'Today', module: 'dashboard', defaultSize: 'sm' },
  TodayOverviewWidget,
)
registerWidget(
  { id: 'quick-expense', title: 'Quick Expense', module: 'finance', defaultSize: 'sm' },
  QuickExpenseWidget,
)
registerWidget(
  { id: 'recent-transactions', title: 'Recent Transactions', module: 'finance', defaultSize: 'md' },
  RecentTransactionsWidget,
)
registerWidget(
  { id: 'habit-streak', title: 'Habit Streak', module: 'habits', defaultSize: 'sm' },
  HabitStreakWidget,
)
registerWidget(
  { id: 'habit-activity', title: 'Activity', module: 'habits', defaultSize: 'md' },
  HabitActivityWidget,
)