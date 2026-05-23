# Offline & Sync

## PWA

- Web manifest (vite-plugin-pwa)
- Service worker (Workbox, auto-update)
- Offline fallback page
- App icons

## Offline-Safe Operations

- Finance transaction drafts
- Habit check-ins
- Dashboard layout changes
- Widget configuration

## Server-Authoritative

- Auth/sessions
- Permissions
- Final account balances
- External integrations

## IndexedDB Queue

All offline mutations stored in IndexedDB with:
- `clientMutationId` (UUID)
- `schemaVersion` (for migration)
- Zod validation on read
- Retry with exponential backoff

## Sync Protocol

- `POST /api/v1/sync/push` -- send queued mutations
- `POST /api/v1/sync/pull` -- fetch latest state
- `GET /api/v1/sync/status` -- check sync health

## Conflict Strategy

- Finance: client-created records synced, server-authoritative balances
- Habits: optimistic check-ins, server-authoritative streaks
- Dashboard: last-write-wins with revision tracking