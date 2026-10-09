package rbac

import "testing"

func TestRoleHierarchy(t *testing.T) {
	if !AtLeast(Owner, Viewer) || !AtLeast(Admin, Member) || !AtLeast(Member, Member) {
		t.Fatal("expected hierarchy to hold")
	}
	if AtLeast(Viewer, Member) || AtLeast(Member, Admin) || AtLeast(Role("ghost"), Viewer) {
		t.Fatal("privilege escalation allowed")
	}
}

func TestWriteAndAdminGates(t *testing.T) {
	for _, role := range []Role{Owner, Admin, Member} {
		if !CanWrite(role) {
			t.Fatalf("%s should write", role)
		}
	}
	if CanWrite(Viewer) {
		t.Fatal("viewer must be read-only")
	}
	for _, role := range []Role{Owner, Admin} {
		if !CanAdmin(role) {
			t.Fatalf("%s should admin", role)
		}
	}
	if CanAdmin(Member) || CanAdmin(Viewer) {
		t.Fatal("member/viewer must not admin")
	}
}

func TestParseRejectsUnknown(t *testing.T) {
	if _, ok := Parse("superadmin"); ok {
		t.Fatal("unknown role parsed")
	}
	if _, ok := Parse(""); ok {
		t.Fatal("empty role parsed")
	}
}

func TestResolveOrg(t *testing.T) {
	memberships := []Membership{{OrgID: "org-a", Role: Member}}
	if _, err := ResolveOrg(memberships, ""); err != ErrOrgRequired {
		t.Fatalf("missing org err = %v", err)
	}
	if _, err := ResolveOrg(memberships, "org-b"); err != ErrNoAccess {
		t.Fatalf("foreign org err = %v", err)
	}
	got, err := ResolveOrg(memberships, "org-a")
	if err != nil || got.Role != Member {
		t.Fatalf("own org err = %v", err)
	}
}
