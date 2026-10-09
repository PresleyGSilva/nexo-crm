-- 001_init.sql — Nexo CRM base schema (multi-tenant: every row carries org_id).
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS orgs (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  slug       TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS memberships (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  org_id  UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  role    TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, org_id)
);
CREATE INDEX IF NOT EXISTS memberships_org_idx ON memberships (org_id);

CREATE TABLE IF NOT EXISTS contacts (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id     UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  email      TEXT,
  phone      TEXT,
  company    TEXT,
  position   TEXT,
  status     TEXT NOT NULL DEFAULT 'active',
  notes      TEXT,
  created_by UUID REFERENCES users (id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS contacts_org_idx ON contacts (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS contacts_org_search_idx ON contacts (org_id, name, email, company);

CREATE TABLE IF NOT EXISTS deals (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id         UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  contact_id     UUID REFERENCES contacts (id) ON DELETE SET NULL,
  title          TEXT NOT NULL,
  description    TEXT,
  value_cents    BIGINT NOT NULL DEFAULT 0,
  stage          TEXT NOT NULL DEFAULT 'prospecting'
                 CHECK (stage IN ('prospecting', 'qualified', 'proposal', 'negotiation', 'won', 'lost')),
  expected_close DATE,
  created_by     UUID REFERENCES users (id) ON DELETE SET NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS deals_org_idx ON deals (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS deals_org_stage_idx ON deals (org_id, stage);

CREATE TABLE IF NOT EXISTS tasks (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id      UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  description TEXT,
  status      TEXT NOT NULL DEFAULT 'pending'
              CHECK (status IN ('pending', 'in_progress', 'done', 'cancelled')),
  due_date    DATE,
  assigned_to UUID REFERENCES users (id) ON DELETE SET NULL,
  contact_id  UUID REFERENCES contacts (id) ON DELETE SET NULL,
  deal_id     UUID REFERENCES deals (id) ON DELETE SET NULL,
  created_by  UUID REFERENCES users (id) ON DELETE SET NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tasks_org_idx ON tasks (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS tasks_org_status_idx ON tasks (org_id, status);

CREATE TABLE IF NOT EXISTS interactions (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id     UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts (id) ON DELETE CASCADE,
  deal_id    UUID REFERENCES deals (id) ON DELETE CASCADE,
  type       TEXT NOT NULL CHECK (type IN ('call', 'email', 'meeting', 'note', 'whatsapp')),
  content    TEXT NOT NULL,
  created_by UUID REFERENCES users (id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT interactions_target CHECK (contact_id IS NOT NULL OR deal_id IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS interactions_org_idx ON interactions (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS interactions_contact_idx ON interactions (contact_id);
CREATE INDEX IF NOT EXISTS interactions_deal_idx ON interactions (deal_id);

CREATE TABLE IF NOT EXISTS audit_logs (
  id         BIGSERIAL PRIMARY KEY,
  org_id     UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  user_id    UUID REFERENCES users (id) ON DELETE SET NULL,
  action     TEXT NOT NULL,
  entity     TEXT NOT NULL,
  entity_id  TEXT,
  metadata   JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_logs_org_idx ON audit_logs (org_id, id DESC);
CREATE INDEX IF NOT EXISTS audit_logs_org_entity_idx ON audit_logs (org_id, entity, id DESC);

CREATE TABLE IF NOT EXISTS org_settings (
  org_id     UUID PRIMARY KEY REFERENCES orgs (id) ON DELETE CASCADE,
  settings   JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Keep updated_at fresh on row changes.
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_orgs_updated ON orgs;
CREATE TRIGGER trg_orgs_updated BEFORE UPDATE ON orgs
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_users_updated ON users;
CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_contacts_updated ON contacts;
CREATE TRIGGER trg_contacts_updated BEFORE UPDATE ON contacts
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_deals_updated ON deals;
CREATE TRIGGER trg_deals_updated BEFORE UPDATE ON deals
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_tasks_updated ON tasks;
CREATE TRIGGER trg_tasks_updated BEFORE UPDATE ON tasks
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
DROP TRIGGER IF EXISTS trg_settings_updated ON org_settings;
CREATE TRIGGER trg_settings_updated BEFORE UPDATE ON org_settings
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
