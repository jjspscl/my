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
- offline fallback
- cached app shell
- update notification
- network status UI

Offline-safe:
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
