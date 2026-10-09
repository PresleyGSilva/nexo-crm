# Nexo CRM — CRM SaaS multi-tenant (Go + Next.js)

[![CI](https://github.com/PresleyGSilva/nexo-crm/actions/workflows/ci.yml/badge.svg)](https://github.com/PresleyGSilva/nexo-crm/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](frontend/package.json)
[![Postgres](https://img.shields.io/badge/Postgres-16-4169E1?logo=postgresql&logoColor=white)](docker-compose.yml)
[![Licença: MIT](https://img.shields.io/badge/Licen%C3%A7a-MIT-green.svg)](LICENSE)

> 🇺🇸 English version: [README.md](README.md)

Um CRM multi-tenant MVP: contatos, funil de deals, tarefas, timeline de
interações, dashboard, audit log e settings por organização — servidos por
uma API Go com isolamento real de tenants (Row-Level Security no Postgres)
e um frontend Next.js.

## O problema

Pequenas empresas de serviços vendem por planilhas e threads de chat:
sem funil compartilhado, sem histórico de quem falou com quem, sem controle
de acesso. Os CRMs existentes ou são brinquedos single-tenant (uma query
vazada expõe todos os clientes) ou plataformas pesadas que levam semanas
para adotar.

O Nexo CRM responde a uma pergunta mais estreita: **qual é o mínimo de
código para entregar um CRM onde o isolamento de tenants é garantido pelo
próprio banco de dados — e não por lembrar de escrever
`WHERE org_id = …` em todo handler?**

## Funcionalidades

- **Auth** — registro/login, access token JWT curto (15 min) + refresh
  opaco rotativo em cookie `httpOnly` (7 dias); recuperação de senha com
  anti-enumeração (sempre 200, token vai para o log — plugue um provedor
  de e-mail aqui).
- **Multi-tenancy à prova de bugs** — toda requisição leva `X-Org-ID`, o
  middleware valida o membership, fixa `app.org_id` numa conexão dedicada,
  e as políticas RLS do Postgres rejeitam linhas de outro tenant mesmo se
  uma query esquecer o filtro.
- **RBAC** — `owner / admin / member / viewer`; viewers leem, members
  escrevem, admins gerenciam settings/auditoria/exclusões.
- **Contatos** — busca (`q`), filtro por status, paginação.
- **Deals** — estágios do funil com endpoint explícito e auditado de
  transição de `stage`; valores em **centavos** (`value_cents`).
- **Tarefas e interações** — prazos, status, timeline por contato/deal.
- **Dashboard** — contadores, totais do funil por estágio, taxa de ganho.
- **Audit log** — toda mutação registrada com ator + org.
- **Settings da org** — documento JSONB com semântica de merge-patch.
- **Contrato da API** — [`openapi.yaml`](openapi.yaml), também servido ao
  vivo em `GET /openapi.yaml`; listas em `{data, page, per_page, total}`,
  erros em `{error: {code, message}}`.
- **Frontend** — login/registro, dashboard, contatos, deals, tarefas,
  auditoria, settings, com fallback de dados mock para trabalhar na UI
  sem a API.

## Stack

| Camada   | Escolha | Por quê |
|----------|---------|---------|
| API      | Go 1.23, router Chi, pgx v5 | HTTP estilo stdlib, um ótimo driver Postgres, poucas dependências |
| Banco    | Postgres 16, migrations SQL versionadas, RLS | o banco impõe o tenancy; migrations em `migrations/` |
| AuthN/Z  | JWT (access) + refresh opaco, bcrypt, stores em memória | bearer curto, renovação segura contra XSS; troque por Redis depois |
| Frontend | Next.js 15 App Router, React 19, Tailwind | páginas SSR, deploy simples, estilização utilitária |
| Contrato | OpenAPI 3.0 | frontend/backend evoluem em paralelo contra um arquivo |
| CI       | GitHub Actions (`go vet` + `go test`, `tsc` + `next build`) | main sempre verde |

## Arquitetura e decisões

```
Navegador → Next.js (web:3000) → API Go (api:8080/api/v1) → Postgres 16 (RLS)
```

Caminho da requisição: `middleware de auth (JWT)` → `middleware de org
(X-Org-ID + membership → SET app.org_id)` → `handler (filtro explícito
por org_id)` → `política RLS (org_id = current_setting('app.org_id'))`.

Detalhes e diagramas: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
Trade-offs principais: [`docs/DECISIONS.md`](docs/DECISIONS.md) — Go
stdlib-first, `X-Org-ID` explícito em vez de inferir tenant do JWT, RLS +
filtro na app (defesa em profundidade), cookies de refresh `httpOnly`,
OpenAPI-first.

## Como rodar

Pré-requisitos: Docker + Compose, Go 1.23+, Node 20+.

```bash
# 1. Stack completa (postgres + api + web)
docker compose up --build
# web → http://localhost:3000 · api → http://localhost:8080/api/v1

# 2. Ou rode local contra um Postgres descartável
docker run -d --name crm-postgres \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=crm \
  -p 5432:5432 postgres:16-alpine

cd backend
cp ../.env.example ../.env   # opcional; o backend migra + semeia sozinho
go run ./cmd/api             # :8080

cd ../frontend
cp .env.example .env.local
npm install && npm run dev   # :3000
```

Login demo (dados fictícios semeados — org **Acme Demo**, 20 contatos,
15 deals, tarefas, interações):

| Campo | Valor |
|-------|-------|
| E-mail | `demo@nexo.local` |
| Senha | `Demo1234!` |

```bash
curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@nexo.local","password":"Demo1234!"}'
```

## Configuração

| Var | Padrão | Para quê |
|-----|--------|----------|
| `PORT` | `8080` | porta da API |
| `DATABASE_URL` | postgres local | pool pgx + runner de migrations |
| `JWT_SECRET` | `dev-secret-change-me` | HMAC dos access tokens (**troque em prod**) |
| `JWT_ACCESS_TTL` | `15m` | tempo de vida do access token |
| `JWT_REFRESH_TTL` | `168h` | tempo de vida do refresh token |
| `COOKIE_SECURE` | `false` | `true` atrás de HTTPS |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080/api/v1` | base da API para o frontend |

## Referência da API

Base `http://localhost:8080/api/v1`. Esquema completo: [`openapi.yaml`](openapi.yaml).

| Método | Rota | Notas |
|--------|------|-------|
| POST | `/auth/register\|login\|refresh\|forgot\|reset` | sessão/recuperação (públicas) |
| GET | `/auth/me` | identidade + orgs |
| GET/POST | `/contacts` | `q, page, per_page, status` |
| GET/PATCH/DELETE | `/contacts/:id` | |
| GET/POST | `/deals` | `q, stage, page, per_page` |
| GET/PATCH/DELETE | `/deals/:id` | |
| PATCH | `/deals/:id/stage` | `{stage}`, auditado |
| GET/POST | `/tasks` | `q, status, page, per_page` |
| GET/PATCH/DELETE | `/tasks/:id` | |
| GET/POST | `/interactions` | `contact_id, deal_id, page, per_page` |
| GET | `/interactions/:id` | |
| GET | `/dashboard/summary` | contadores, funil por estágio, win rate |
| GET | `/audit-logs` | `entity, page, per_page` |
| GET/PATCH | `/settings` | merge JSONB (admin+) |

## Testes

```bash
cd backend
go build ./...   # precisa passar
go vet ./...     # precisa passar
go test ./...    # unitários + isolamento de middleware (sem banco)

# Com Postgres: exercita o caminho real do RLS
DATABASE_URL=postgres://postgres:postgres@localhost:5432/crm?sslmode=disable go test ./...

cd ../frontend
npm run lint     # tsc --noEmit
npm run build    # build de produção
```

O que a suíte prova: `TestCrossTenantIsolation` (org estrangeira →
`ErrNoAccess`, viewer em endpoint de escrita → 403), round-trip/expiry
de JWT, rotação de refresh, hierarquia RBAC, políticas RLS aplicadas.

## Estrutura do projeto

```
crm-saas/
├── backend/cmd/api/      # entrypoint: config, pool, migrate, serve
├── backend/internal/
│   ├── auth/             # bcrypt, JWT access, stores de refresh/reset
│   ├── rbac/             # roles + resolução de org
│   ├── config/           # env
│   ├── db/               # pool pgx + runner de migrations versionadas
│   └── httpapi/          # router, middlewares, handlers
├── frontend/app/         # páginas App Router (dashboard, contatos, deals…)
├── frontend/lib/         # client da API, contexto de auth, tipos
├── frontend/components/  # shell, sidebar, widgets
├── migrations/           # 001 schema · 002 RLS · 003 seed demo · 004 cascade
├── docs/                 # ARCHITECTURE.md · DECISIONS.md
└── openapi.yaml          # contrato da API
```

## Licença

MIT — veja [LICENSE](LICENSE).
