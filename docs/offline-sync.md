# Offline & Sync

## Current status

Offline support exists, but it is partial infrastructure rather than a fully integrated sync platform.

Document current behavior, not roadmap-only behavior.

## Implemented today

### PWA shell

- web manifest via `vite-plugin-pwa`
- service worker with `registerType: 'autoUpdate'`
- Workbox runtime caching for `/api/v1/*`
- `NetworkFirst` API cache with 5s network timeout and 10-minute expiry
- Auth endpoints are excluded from runtime caching.
- API cache is purged after successful logout.

### Client-side offline infra

Files live under `apps/web/src/shared/sync/`.

Current pieces:

- `network-status.ts` — Zustand online/offline store
- `mutation-queue.ts` — IndexedDB queue via `idb-keyval`
- `offline-mutate.ts` — helper to queue failed/offline mutations
- `sync-engine.ts` — replay engine + queue drain scheduling

### Queue behavior

Queued mutation fields:

- `id`
- `schemaVersion`
- `method`
- `url`
- `body`
- `createdAt`
- `retries`
- `maxRetries`

Behavior:

- queue entries validated with Zod on read
- invalid/corrupt entries are discarded
- drain runs on startup
- drain runs when connectivity returns
- drain also runs every 30 seconds
- replay re-sends the original request with cookies and `X-CSRF-Token`
- 4xx responses are treated as permanent failure and discarded
- 5xx/network failures are retried until `maxRetries`

## Important limitations

These are **not** currently implemented end to end:

- dedicated `/api/v1/sync/push`
- dedicated `/api/v1/sync/pull`
- dedicated `/api/v1/sync/status`
- conflict-resolution protocol between client/server revisions
- universal adoption of queue-backed mutations across all features

`offlineMutate()` currently exists as infrastructure, but feature hooks are not yet universally wired through it.

## Server-authoritative areas

Even with offline support, server truth still matters for:

- auth/session validity
- final wallet/account balances
- habit streak derivation
- any future multi-device reconciliation

## Documentation rule

If queue integration expands or a real sync API lands, update this doc to separate:

- current implementation
- future roadmap
