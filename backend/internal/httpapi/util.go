package httpapi

import (
	"regexp"
	"strings"
	"time"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isUUID reports whether s looks like a UUID string.
func isUUID(s string) bool {
	return uuidRe.MatchString(s)
}

// strptr dereferences s, returning "" for nil.
func strptr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// parseDate accepts YYYY-MM-DD or RFC3339 and returns the date at UTC midnight.
func parseDate(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC().Truncate(24 * time.Hour), true
	}
	return time.Time{}, false
}

// formatDate renders t as YYYY-MM-DD, or "" when t is nil.
func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}
