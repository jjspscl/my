---
name: tanstack-router-query
description: Enforce TanStack Router for URL state and TanStack Query for server state, loaders, query keys, invalidation, and optimistic updates.
compatibility: opencode
---

# tanstack-router-query

TanStack Router owns:
- route params
- search params
- route context
- loaders
- redirects
- protected layouts
- pending states
- error states

TanStack Query owns:
- server data
- cache
- mutations
- invalidation
- optimistic updates
- retries
- background refetch

Never store server state in Zustand.
Never store URL state in Zustand.
Never manually parse `window.location`.

Use route loaders with:
- Zod search validation
- `queryClient.ensureQueryData`
- matching query hooks in components

Each feature must have query key factories.
No scattered raw query keys.
