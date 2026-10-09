# Nexo CRM

Multi-tenant CRM SaaS MVP (Go + Next.js). Public repo, push-ready.

## Quickstart

```bash
cp .env.example .env
docker compose up --build
```

- API: http://localhost:8080/api/v1
- Web: http://localhost:3000
- OpenAPI: [`openapi.yaml`](openapi.yaml)

## Layout

- `backend/` — Go API (`cmd/api`, `internal/{handler,middleware,model,store}`)
- `frontend/` — Next.js app (`app`, `components`, `lib`)
- `migrations/` — Postgres SQL
- `docs/` — `ARCHITECTURE.md`, `DECISIONS.md`
