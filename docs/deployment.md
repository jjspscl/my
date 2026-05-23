# Deployment

## Production Binary

```bash
mise run build
./bin/my
```

The binary serves both API and frontend on a single port (default 8080).

## Environment Variables

See `.env.example` for all available configuration.

## Docker

```dockerfile
# Build stage
FROM golang:1.26 AS go-builder
FROM node:22 AS web-builder

# ... build frontend, copy to Go embed dir, build Go binary

# Runtime
FROM gcr.io/distroless/base
COPY --from=go-builder /app/bin/my /my
EXPOSE 8080
CMD ["/my"]
```

## Targets

- Docker / Docker Compose
- Kubernetes
- Fly.io
- Railway
- Render
- AWS ECS
- GCP Cloud Run

## Database

- **Development**: Embedded libSQL (file:my_dev.db)
- **Production**: Turso cloud or self-hosted sqld
- **Redis**: Session cache, rate limiting