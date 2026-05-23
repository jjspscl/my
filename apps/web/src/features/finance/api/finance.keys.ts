export const financeKeys = {
  all: ['finance'] as const,
  transactions: () => [...financeKeys.all, 'transactions'] as const,
  transactionList: (filters?: Record<string, string>) =>
    [...financeKeys.transactions(), 'list', filters] as const,
  todayTotal: () => [...financeKeys.transactions(), 'today-total'] as const,
}