# Frontend

## Stack

- React 19
- Vite 8
- TypeScript 6 (strict)
- TanStack Router (file-based routing)
- TanStack Query (server state)
- Zod 4 (schema-first contracts)
- React Hook Form + zodResolver
- Zustand (ephemeral UI state only)
- shadcn/ui + Tailwind CSS v4 + Radix + lucide-react

## Rules

1. All application data types must be inferred from Zod schemas
2. No `interface` or `type` for API data -- use `z.infer<>`
3. No `response.json() as T` -- always parse with Zod
4. Server state lives in TanStack Query, not Zustand
5. URL state lives in TanStack Router, not Zustand
6. Each feature has query key factories
7. Route search params validate through Zod

## Directory Structure

```
src/
  routes/          TanStack Router file-based routes
  features/        Feature modules (schemas, api, components, hooks)
  components/      Shared components (ui, layout, widgets, feedback, nav)
  shared/          Utilities, types, hooks, API client
  stores/          Zustand stores (ephemeral UI only)
  styles/          Global CSS
```

## Scripts

| Script | Purpose |
|---|---|
| `pnpm dev` | Vite dev server with HMR |
| `pnpm build` | TypeScript check + Vite production build |
| `pnpm typecheck` | TypeScript --noEmit |
| `pnpm lint` | ESLint |
| `pnpm test` | Vitest |
| `pnpm route:generate` | TanStack Router code generation |