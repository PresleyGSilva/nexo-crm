# Nexo CRM — Multi-tenant SaaS CRM (Go + Next.js)

[![CI](https://github.com/PresleyGSilva/nexo-crm/actions/workflows/ci.yml/badge.svg)](https://github.com/PresleyGSilva/nexo-crm/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](frontend/package.json)
[![Postgres](https://img.shields.io/badge/Postgres-16-4169E1?logo=postgresql&logoColor=white)](docker-compose.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 🇧🇷 Versão em português: [README.pt-BR.md](README.pt-BR.md)

A multi-tenant CRM MVP: contacts, deal pipeline, tasks, interaction timeline,
dashboard, audit log and per-org settings — served by a Go API with real
tenant isolation (Postgres Row-Level Security) and a Next.js frontend.

## The problem

Small service businesses run sales out of spreadsheets and chat threads:
no shared pipeline, no history of who talked to whom, no access control.
Existing CRMs are either single-tenant toys (one leaked query exposes every
customer) or heavyweight platforms that take weeks to adopt.

Nexo CRM answers a narrower question: **how little code does it take to ship
a CRM where tenant isolation is enforced by the database itself, not just by
remembering `WHERE org_id = …` in every handler?**

## Features

- **Auth** — register/login, short-lived JWT access token (15 min) + rotating
  opaque refresh token in an `httpOnly` cookie (7 days); password
  forgot/reset with anti-enumeration (always 200, token goes to server log —
  plug an email provider here).
- **Multi-tenancy that survives bugs** — every request carries `X-Org-ID`,
  the middleware verifies membership, pins `app.org_id` on a dedicated
  connection, and Postgres RLS policies reject cross-tenant rows even if a
  query forgets its filter.
- **RBAC** — `owner / admin / member / viewer`; viewers read, members
  write, admins manage settings/audit/deletes.
- **Contacts** — search (`q`), status filter, pagination.
- **Deals** — pipeline stages with an explicit audited `stage` transition
  endpoint; money stored in **cents** (`value_cents`).
- **Tasks & interactions** — due dates, status, per-contact/per-deal timeline.
- **Dashboard** — counts, pipeline totals per stage, win rate.
- **Audit log** — every mutation recorded with actor + org.
- **Org settings** — JSONB document with merge-patch semantics.
- **API contract** — [`openapi.yaml`](openapi.yaml), also served live at
  `GET /openapi.yaml`; list responses use `{data, page, per_page, total}`,
  errors use `{error: {code, message}}`.
- **Frontend** — login/register, dashboard, contacts, deals, tasks, audit,
  settings, with a mock-data fallback for UI work without the API.

## Screenshots (real app, local run with seed data)

| Login | Dashboard |
|---|---|
| ![Login](docs/screenshots/01-login.png) | ![Dashboard](docs/screenshots/02-dashboard.png) |

| Contacts | Deals Kanban |
|---|---|
| ![Contacts](docs/screenshots/03-contacts.png) | ![Deals](docs/screenshots/04-deals.png) |

| Tasks | Audit log |
|---|---|
| ![Tasks](docs/screenshots/05-tasks.png) | ![Audit](docs/screenshots/06-audit.png) |

## Stack

| Layer    | Choice | Why |
|----------|--------|-----|
| API      | Go 1.23, Chi router, pgx v5 | stdlib-style HTTP, one great Postgres driver, tiny dependency tree |
| DB       | Postgres 16, versioned SQL migrations, RLS | the database enforces tenancy; migrations live in `migrations/` |
| AuthN/Z  | JWT (access) + opaque refresh, bcrypt, in-memory stores | short-lived bearer, XSS-safe renewal; swap stores for Redis later |
| Frontend | Next.js 15 App Router, React 19, Tailwind | SSR pages, simple deploy, utility styling |
| Contract | OpenAPI 3.0 | frontend/backend develop in parallel against one file |
| CI       | GitHub Actions (`go vet` + `go test`, `tsc` + `next build`) | green main, always |

## Architecture & decisions

```
Browser → Next.js (web:3000) → Go API (api:8080/api/v1) → Postgres 16 (RLS)
```

Request path: `auth middleware (JWT)` → `org middleware (X-Org-ID +
membership → SET app.org_id)` → `handler (explicit org_id filter)` →
`RLS policy (org_id = current_setting('app.org_id'))`.

Details and diagrams: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
Key trade-offs: [`docs/DECISIONS.md`](docs/DECISIONS.md) — stdlib-first Go,
explicit `X-Org-ID` over JWT tenant inference, RLS + app filter
(defense in depth), `httpOnly` refresh cookies, OpenAPI-first.

## Quickstart

Prerequisites: Docker + Compose, Go 1.23+, Node 20+.

```bash
# 1. Full stack (postgres + api + web)
docker compose up --build
# web → http://localhost:3000 · api → http://localhost:8080/api/v1

# 2. Or run locally against a throwaway Postgres
docker run -d --name crm-postgres \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=crm \
  -p 5432:5432 postgres:16-alpine

cd backend
cp ../.env.example ../.env   # optional; backend auto-migrates + seeds
go run ./cmd/api             # :8080

cd ../frontend
cp .env.example .env.local
npm install && npm run dev   # :3000
```

Demo login (seeded, fictitious data — org **Acme Demo**, 20 contacts,
15 deals, tasks, interactions):

| Field | Value |
|-------|-------|
| Email | `demo@nexo.local` |
| Password | `Demo1234!` |

```bash
curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@nexo.local","password":"Demo1234!"}'
```

## Configuration

| Var | Default | Purpose |
|-----|---------|---------|
| `PORT` | `8080` | API listen port |
| `DATABASE_URL` | postgres local | pgx pool + migration runner |
| `JWT_SECRET` | `dev-secret-change-me` | HMAC for access tokens (**set in prod**) |
| `JWT_ACCESS_TTL` | `15m` | access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | refresh token lifetime |
| `COOKIE_SECURE` | `false` | set `true` behind HTTPS |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080/api/v1` | API base for the frontend |

## API reference

Base `http://localhost:8080/api/v1`. Full schema: [`openapi.yaml`](openapi.yaml).

| Method | Route | Notes |
|--------|-------|-------|
| POST | `/auth/register\|login\|refresh\|forgot\|reset` | public session/recovery |
| GET | `/auth/me` | identity + orgs |
| GET/POST | `/contacts` | `q, page, per_page, status` |
| GET/PATCH/DELETE | `/contacts/:id` | |
| GET/POST | `/deals` | `q, stage, page, per_page` |
| GET/PATCH/DELETE | `/deals/:id` | |
| PATCH | `/deals/:id/stage` | `{stage}`, audited |
| GET/POST | `/tasks` | `q, status, page, per_page` |
| GET/PATCH/DELETE | `/tasks/:id` | |
| GET/POST | `/interactions` | `contact_id, deal_id, page, per_page` |
| GET | `/interactions/:id` | |
| GET | `/dashboard/summary` | counts, pipeline by stage, win rate |
| GET | `/audit-logs` | `entity, page, per_page` |
| GET/PATCH | `/settings` | JSONB merge (admin+) |

## Tests

```bash
cd backend
go build ./...   # must pass
go vet ./...     # must pass
go test ./...    # unit + middleware isolation (no DB needed)

# With Postgres: exercises the real RLS path
DATABASE_URL=postgres://postgres:postgres@localhost:5432/crm?sslmode=disable go test ./...

cd ../frontend
npm run lint     # tsc --noEmit
npm run build    # production build
```

What the suite proves: `TestCrossTenantIsolation` (foreign-org resolve →
`ErrNoAccess`, viewer on a write endpoint → 403), JWT round-trip/expiry,
refresh rotation, RBAC hierarchy, RLS policies applied.

## Project structure

```
crm-saas/
├── backend/cmd/api/      # entrypoint: config, pool, migrate, serve
├── backend/internal/
│   ├── auth/             # bcrypt, JWT access, refresh/reset stores
│   ├── rbac/             # roles + org resolution
│   ├── config/           # env
│   ├── db/               # pgx pool + versioned migration runner
│   └── httpapi/          # router, middlewares, handlers
├── frontend/app/         # App Router pages (dashboard, contacts, deals…)
├── frontend/lib/         # API client, auth context, types
├── frontend/components/  # shell, sidebar, widgets
├── migrations/           # 001 schema · 002 RLS · 003 demo seed · 004 cascade
├── docs/                 # ARCHITECTURE.md · DECISIONS.md
└── openapi.yaml          # API contract
```

## License

MIT — see [LICENSE](LICENSE).
