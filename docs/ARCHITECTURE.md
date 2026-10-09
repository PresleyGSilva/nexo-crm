# Nexo CRM — Architecture

Multi-tenant CRM SaaS MVP. Monorepo: `backend/` (Go stdlib API),
`frontend/` (Next.js), `migrations/` (Postgres SQL), `docs/`.

## Tenancy

- Every tenant table carries `org_id UUID`.
- Request tenant comes from `X-Org-ID` header; middleware checks
  `memberships(user_id, org_id)` before any query.
- Postgres RLS enabled per table (`USING (org_id = current_setting('app.org_id')::uuid)`);
  backend sets `app.org_id` per transaction AND filters by `org_id` explicitly (defense in depth).

## Auth

- JWT access token (Bearer, short TTL) + refresh token in httpOnly cookie.
- `POST /auth/register|login|refresh|forgot|reset`.

## API

- Base `http://localhost:8080/api/v1`, contract in `/openapi.yaml`.
- Resources: contacts (q/page/per_page), deals (+ stage move),
  tasks, interactions (read), dashboard/summary, audit-logs, settings.

## Data

- Postgres 16 via docker-compose (`api`, `web`, `postgres16`).
- Seed (fictitious): 1 demo org, 20 contacts, 15 deals.
