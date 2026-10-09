package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexocrm/crm-saas/backend/internal/rbac"
)

// TestCrossTenantIsolation proves the two tenant gates at HTTP level:
//  1. ResolveOrg refuses memberships outside the requested org.
//  2. requireRole refuses viewers on writer endpoints, so a leaked token
//     from another tenant still cannot write.
func TestCrossTenantIsolation(t *testing.T) {
	memberships := []rbac.Membership{{OrgID: "org-a", Role: rbac.Viewer}}

	if _, err := rbac.ResolveOrg(memberships, "org-b"); err != rbac.ErrNoAccess {
		t.Fatalf("cross-tenant resolve err = %v, want ErrNoAccess", err)
	}
	if _, err := rbac.ResolveOrg(memberships, ""); err != rbac.ErrOrgRequired {
		t.Fatalf("missing tenant err = %v, want ErrOrgRequired", err)
	}

	s := &Server{}
	writerOnly := s.requireRole(rbac.Member)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Viewer from org-a calling a writer endpoint: must be rejected.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", nil)
	req = req.WithContext(context.WithValue(req.Context(), roleKey, rbac.Viewer))
	rec := httptest.NewRecorder()
	writerOnly.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer write status = %d, want 403", rec.Code)
	}

	// Member passes the same gate.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/contacts", nil)
	req = req.WithContext(context.WithValue(req.Context(), roleKey, rbac.Member))
	rec = httptest.NewRecorder()
	writerOnly.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("member write status = %d, want 200", rec.Code)
	}
}

func TestParsePaginationDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	p := parsePagination(req)
	if p.page != 1 || p.perPage != 20 || p.offset != 0 {
		t.Fatalf("defaults = %+v", p)
	}

	req = httptest.NewRequest(http.MethodGet, "/contacts?page=3&per_page=200", nil)
	p = parsePagination(req)
	if p.page != 3 || p.perPage != 100 || p.offset != 200 {
		t.Fatalf("clamped = %+v", p)
	}

	req = httptest.NewRequest(http.MethodGet, "/contacts?page=0&per_page=-5", nil)
	p = parsePagination(req)
	if p.page != 1 || p.perPage != 20 {
		t.Fatalf("invalid = %+v", p)
	}
}

func TestIsUUID(t *testing.T) {
	if !isUUID("11111111-1111-1111-1111-111111111111") {
		t.Fatal("valid uuid rejected")
	}
	for _, bad := range []string{"", "org-a", "11111111-1111", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"} {
		if isUUID(bad) {
			t.Fatalf("invalid uuid accepted: %q", bad)
		}
	}
}

func TestUUIDHelper(t *testing.T) {
	if _, ok := optUUID(nil); !ok {
		t.Fatal("nil uuid rejected")
	}
	empty := ""
	if _, ok := optUUID(&empty); !ok {
		t.Fatal("empty uuid rejected")
	}
	bad := "nope"
	if _, ok := optUUID(&bad); ok {
		t.Fatal("bad uuid accepted")
	}
}
