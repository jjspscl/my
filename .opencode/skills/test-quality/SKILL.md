---
name: test-quality
description: Add focused backend, frontend, schema, route, mutation, viewport, and offline tests without over-testing implementation details.
compatibility: opencode
---

# test-quality

Backend tests:
- table-driven unit tests
- domain tests
- application service tests
- repository integration tests
- httptest handlers
- sync conflict tests
- auth/security tests

Frontend tests:
- Vitest
- React Testing Library
- Playwright
- MSW when useful

Test:
- route guards
- query hooks
- mutation invalidation
- optimistic rollback
- Zod schemas
- forms
- mobile/tablet/desktop layouts
- offline fallback

Do not test shadcn internals.
Prefer behavior tests.
