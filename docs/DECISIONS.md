# Nexo CRM — Decisions

1. **Go stdlib, no framework** — MVP stays dependency-light; `net/http`
   router is enough, easy to maintain.
2. **Next.js App Router** — public MVP needs SSR + simple deploy.
3. **X-Org-ID header + membership check** — explicit tenant selection,
   verified server-side; no tenant inference from JWT alone.
4. **RLS + app-level org_id filter** — belt and suspenders for tenant isolation.
5. **Refresh in httpOnly cookie** — XSS-safe session renewal; access token stays short-lived.
6. **OpenAPI stub first** — frontend/backend parallelize against `/openapi.yaml`.
