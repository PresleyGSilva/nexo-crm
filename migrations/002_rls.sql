-- 002_rls.sql — Row-Level Security: every tenant table is visible only to
-- backends holding that tenant in app.org_id (set per request by the
-- requireOrg middleware via SELECT set_config('app.org_id', ...)).
-- The application ALSO scopes every query with WHERE org_id = $1, so RLS is
-- defense-in-depth for direct/psql access and future restricted roles.
-- NOTE: table owners bypass RLS; the policies below bind least-privilege
-- database roles and any session that sets app.org_id.

ALTER TABLE orgs         ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships  ENABLE ROW LEVEL SECURITY;
ALTER TABLE contacts     ENABLE ROW LEVEL SECURITY;
ALTER TABLE deals        ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks        ENABLE ROW LEVEL SECURITY;
ALTER TABLE interactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs   ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_settings ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS org_isolation ON orgs;
CREATE POLICY org_isolation ON orgs
  FOR ALL USING (id::text = current_setting('app.org_id', true))
  WITH CHECK (id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON memberships;
CREATE POLICY org_isolation ON memberships
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON contacts;
CREATE POLICY org_isolation ON contacts
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON deals;
CREATE POLICY org_isolation ON deals
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON tasks;
CREATE POLICY org_isolation ON tasks
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON interactions;
CREATE POLICY org_isolation ON interactions
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON audit_logs;
CREATE POLICY org_isolation ON audit_logs
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));

DROP POLICY IF EXISTS org_isolation ON org_settings;
CREATE POLICY org_isolation ON org_settings
  FOR ALL USING (org_id::text = current_setting('app.org_id', true))
  WITH CHECK (org_id::text = current_setting('app.org_id', true));
