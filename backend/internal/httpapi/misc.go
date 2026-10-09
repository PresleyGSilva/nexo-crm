package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleHealth is an unauthenticated liveness probe.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleOpenAPI serves the repository-level openapi.yaml contract.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	for _, candidate := range []string{"../openapi.yaml", "./openapi.yaml", "openapi.yaml", "../../openapi.yaml"} {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		body, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}
	writeErr(w, http.StatusNotFound, "not_found", "openapi.yaml not found")
}

func slugify(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	prevDash := false
	for _, ch := range lower {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
			b.WriteRune(ch)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "org"
	}
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err == nil {
		slug += "-" + hex.EncodeToString(suffix[:])
	}
	return slug
}

func queryOrgs(ctx context.Context, database DB, userID string) ([]orgJSON, error) {
	rows, err := database.Query(ctx,
		`SELECT o.id, o.name, m.role FROM memberships m
		 JOIN orgs o ON o.id = m.org_id
		 WHERE m.user_id = $1 ORDER BY o.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orgs := []orgJSON{}
	for rows.Next() {
		var o orgJSON
		if err := rows.Scan(&o.ID, &o.Name, &o.Role); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}
