package db

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestRLSPoliciesApplied runs against a real Postgres when DATABASE_URL is
// set: it migrates a scratch database and asserts the org_isolation policy
// exists on every tenant table.
func TestRLSPoliciesApplied(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL unset; skipping live RLS check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	defer pool.Close()

	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "../../../migrations"
	}
	if err := Migrate(ctx, pool, dir); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	tables := []string{"orgs", "memberships", "contacts", "deals", "tasks", "interactions", "audit_logs", "org_settings"}
	for _, table := range tables {
		var count int
		err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM pg_policies WHERE schemaname = 'public' AND tablename = $1 AND policyname = 'org_isolation'`,
			table).Scan(&count)
		if err != nil {
			t.Fatalf("policy lookup %s: %v", table, err)
		}
		if count == 0 {
			t.Fatalf("missing org_isolation policy on %s", table)
		}
	}
}
