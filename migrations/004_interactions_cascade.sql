-- 004_interactions_cascade.sql — Interactions are children of their contact
-- or deal: deleting the parent removes the thread. (ON DELETE SET NULL
-- conflicted with the interactions_target CHECK, making parent deletes fail.)
ALTER TABLE interactions DROP CONSTRAINT IF EXISTS interactions_contact_id_fkey;
ALTER TABLE interactions DROP CONSTRAINT IF EXISTS interactions_deal_id_fkey;
ALTER TABLE interactions
  ADD CONSTRAINT interactions_contact_id_fkey
  FOREIGN KEY (contact_id) REFERENCES contacts (id) ON DELETE CASCADE;
ALTER TABLE interactions
  ADD CONSTRAINT interactions_deal_id_fkey
  FOREIGN KEY (deal_id) REFERENCES deals (id) ON DELETE CASCADE;
