import { QueryClient } from '@tanstack/react-query'

// Single instance shared by the router context, feature hooks, and the sync
// engine (which invalidates queries after a drain without a React context).
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60,
      retry: (failureCount, error: unknown) => {
        const status = (error as { status?: number })?.status
        if (status === 401 || status === 403 || status === 404) return false
        return failureCount < 2
      },
    },
  },
})
