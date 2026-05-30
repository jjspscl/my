export const financeKeys = {
  all: ['finance'] as const,
  transactions: () => [...financeKeys.all, 'transactions'] as const,
  transactionList: (filters?: Record<string, string>) =>
    [...financeKeys.transactions(), 'list', filters] as const,
  todayTotal: () => [...financeKeys.transactions(), 'today-total'] as const,
  budgets: () => [...financeKeys.all, 'budgets'] as const,
  budgetSummary: (month: string) => [...financeKeys.budgets(), 'summary', month] as const,
  bills: () => [...financeKeys.all, 'bills'] as const,
  billList: () => [...financeKeys.bills(), 'list'] as const,
  upcomingBills: (days?: number) => [...financeKeys.bills(), 'upcoming', days ?? 30] as const,
  goals: () => [...financeKeys.all, 'goals'] as const,
  goalList: () => [...financeKeys.goals(), 'list'] as const,
  wallets: () => [...financeKeys.all, 'wallets'] as const,
  walletList: () => [...financeKeys.wallets(), 'list'] as const,
  transfers: () => [...financeKeys.all, 'transfers'] as const,
  transferList: () => [...financeKeys.transfers(), 'list'] as const,
}