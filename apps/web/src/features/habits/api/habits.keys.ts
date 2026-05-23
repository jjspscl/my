export const habitKeys = {
  all: ['habits'] as const,
  list: () => [...habitKeys.all, 'list'] as const,
  completions: (id: string) => [...habitKeys.all, 'completions', id] as const,
  completionsMap: (from?: string, to?: string) => [...habitKeys.all, 'completions-map', from, to] as const,
}