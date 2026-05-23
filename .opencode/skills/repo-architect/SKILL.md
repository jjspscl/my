---
name: repo-architect
description: Enforce the my monorepo architecture, DDD bounded contexts, and clean repository structure.
compatibility: opencode
---

# repo-architect

Protect the architecture of `my`.

Backend contexts live in:

/apps/api/internal/contexts/<context>/
    domain/
    application/
    infrastructure/
    interfaces/http/

Frontend features live in:

/apps/web/src/features/<feature>/
    schemas/
    api/
    components/
    hooks/
    lib/

Do not create global backend folders like:
- /domain
- /service
- /repository
- /handler

Do not put business logic in:
- React route files
- global UI components
- Zustand stores
- shadcn primitive files
