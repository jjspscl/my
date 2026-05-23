---
name: security-review
description: Review auth, API, storage, sync, PWA, finance data, validation, logging, and deployment security.
compatibility: opencode
---

# security-review

Check:
- auth flow
- refresh tokens
- secure cookies
- CSRF when cookie auth enabled
- backend authorization
- rate limits
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

Never allow:
- frontend-only auth enforcement
- secrets in logs
- tokens in unsafe browser storage
- raw SQL injection risk
- unvalidated API payloads
- unvalidated local persisted data
- permissive CORS by default
