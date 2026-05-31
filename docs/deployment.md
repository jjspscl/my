# Deployment

## Production binary

```bash
mise run build
./bin/my
```

The production binary serves both API and frontend on a single port (default `8080`).

## Build flow

`mise run build` does three things:

1. builds the frontend in `apps/web/dist`
2. copies those assets into `apps/api/internal/platform/web/static/`
3. builds the Go binary at `bin/my`

## Environment variables

See `.env.example` for the current supported environment variables.

Key runtime categories:

- API/web ports and URLs
- libSQL/SQLite database settings
- optional Turso remote settings
- Redis session store
- magic-link auth settings
- SMTP settings
- default currency
- MCP-related search/doc API keys

## Docker

No committed production Docker image is currently maintained in this repo.

`docker-compose.yml` is intentionally minimal and currently documents only optional local/libSQL experimentation.

If a production Docker target is added later, update this file to match the real build pipeline.

## Targets

Plausible deployment targets for the single-binary runtime include:

- Docker
- Kubernetes
- Fly.io
- Railway
- Render
- AWS ECS
- GCP Cloud Run

Treat this list as deployment options, not evidence of committed production manifests.

## Database/runtime notes

- **Development DB**: embedded local file (`file:my_dev.db`)
- **Production DB**: embedded SQLite/libSQL or remote Turso/libSQL, depending on env config
- **Redis**: session storage

Rate limiting is not part of the current runtime behavior.
