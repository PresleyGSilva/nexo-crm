// Package rbac defines workspace roles and tenant-scoped access checks.
package rbac

import "errors"

// Role is a membership level inside an organization.
type Role string

// Membership roles, from most to least privileged.
const (
	Owner  Role = "owner"
	Admin  Role = "admin"
	Member Role = "member"
	Viewer Role = "viewer"
)

// ErrNoAccess is returned when a user has no membership in the requested org.
var ErrNoAccess = errors.New("no access to organization")

// ErrOrgRequired is returned when the tenant header is missing.
var ErrOrgRequired = errors.New("organization required")

func rank(role Role) int {
	switch role {
	case Owner:
		return 4
	case Admin:
		return 3
	case Member:
		return 2
	case Viewer:
		return 1
	default:
		return 0
	}
}

// Parse validates a raw role string.
func Parse(raw string) (Role, bool) {
	role := Role(raw)
	switch role {
	case Owner, Admin, Member, Viewer:
		return role, true
	default:
		return "", false
	}
}

// AtLeast reports whether have satisfies the minimum role want.
func AtLeast(have, want Role) bool {
	return rank(have) >= rank(want)
}

// CanWrite reports whether role may create or update CRM records.
func CanWrite(role Role) bool {
	return AtLeast(role, Member)
}

// CanAdmin reports whether role may manage settings, audit and deletes.
func CanAdmin(role Role) bool {
	return AtLeast(role, Admin)
}

// Membership binds a user to an organization with a role.
type Membership struct {
	OrgID string
	Role  Role
}

// ResolveOrg picks the membership matching the requested tenant.
// It is the single choke point for cross-tenant isolation: a request is
// served only when the caller holds an explicit membership in the org
// named by the X-Org-ID header.
func ResolveOrg(memberships []Membership, requested string) (Membership, error) {
	if requested == "" {
		return Membership{}, ErrOrgRequired
	}
	for _, m := range memberships {
		if m.OrgID == requested {
			if _, ok := Parse(string(m.Role)); !ok {
				return Membership{}, ErrNoAccess
			}
			return m, nil
		}
	}
	return Membership{}, ErrNoAccess
}
