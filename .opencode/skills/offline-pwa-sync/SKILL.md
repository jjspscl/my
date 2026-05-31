---
name: offline-pwa-sync
description: Implement PWA installability, offline-tolerant UX, IndexedDB mutation queues, sync contracts, and safe cache behavior.
compatibility: opencode
---

# offline-pwa-sync

Include:
- manifest
- service worker
- icons
- cached app shell when justified
- update notification when justified
- network status UI

Offline-safe candidates:
- finance transaction drafts
- habit check-ins
- dashboard layout drafts
- widget layout changes

Server-authoritative:
- auth
- permissions
- account security
- final finance reconciliation
- external integrations

Every stored object:
- has schemaVersion
- validates with Zod on read
- has migration path or safe discard

Current repo reality:
- IndexedDB mutation queue exists
- replay engine drains on startup, reconnect, and every 30s
- Workbox caches `/api/v1/*` with `NetworkFirst`
- no dedicated `/api/v1/sync/*` backend yet
- `offlineMutate()` infra exists but is not wired through every feature mutation

Do not describe offline sync as fully end-to-end unless the task actually completes those gaps.
