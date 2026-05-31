---
name: security-review
description: Review auth, API, storage, sync, PWA, finance data, validation, logging, and deployment security.
compatibility: opencode
---

# security-review

Check:
- magic-link auth flow
- opaque session cookies
- secure cookie settings
- CSRF when cookie auth enabled
- backend authorization
- request body limits
- input validation
- output encoding
- unsafe local storage
- sensitive cache data
- secret logging
- stack trace leaks
- finance data privacy
- sync idempotency
- replay risks
- optional MCP/token handling in config files

Current repo notes:
- auth is session-cookie based, not JWT/refresh-token based
- Redis stores sessions
- offline replay exists client-side and can re-send queued mutations
- rate limiting is not currently part of runtime behavior, so call it out if missing from a proposed design

Never allow:
- frontend-only auth enforcement
- secrets in logs
- tokens in unsafe browser storage
- raw SQL injection risk
- unvalidated API payloads
- unvalidated local persisted data
- permissive CORS by default
- committed PATs or MCP credentials
