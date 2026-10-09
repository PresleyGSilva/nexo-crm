package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

type stageSummary struct {
	Stage      string `json:"stage"`
	Count      int    `json:"count"`
	TotalCents int64  `json:"total_cents"`
}

// handleDashboardSummary aggregates counts, pipeline totals and win rate.
func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	database := s.db(r)
	org := OrgID(r)

	var contacts, dealsOpen, dealsWon, dealsLost, tasksPending, interactions int
	queries := map[string]struct {
		sql  string
		dest *int
	}{
		"contacts":      {`SELECT COUNT(*) FROM contacts WHERE org_id = $1`, &contacts},
		"deals_open":    {`SELECT COUNT(*) FROM deals WHERE org_id = $1 AND stage NOT IN ('won','lost')`, &dealsOpen},
		"deals_won":     {`SELECT COUNT(*) FROM deals WHERE org_id = $1 AND stage = 'won'`, &dealsWon},
		"deals_lost":    {`SELECT COUNT(*) FROM deals WHERE org_id = $1 AND stage = 'lost'`, &dealsLost},
		"tasks_pending": {`SELECT COUNT(*) FROM tasks WHERE org_id = $1 AND status IN ('pending','in_progress')`, &tasksPending},
		"interactions":  {`SELECT COUNT(*) FROM interactions WHERE org_id = $1`, &interactions},
	}
	for name, q := range queries {
		if err := database.QueryRow(ctx, q.sql, org).Scan(q.dest); err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not load "+name)
			return
		}
	}

	rows, err := database.Query(ctx,
		`SELECT stage, COUNT(*), COALESCE(SUM(value_cents), 0)
		 FROM deals WHERE org_id = $1 GROUP BY stage`, org)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not load pipeline")
		return
	}
	defer rows.Close()

	byStage := []stageSummary{}
	var totalCents int64
	for rows.Next() {
		var s stageSummary
		if err := rows.Scan(&s.Stage, &s.Count, &s.TotalCents); err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not read pipeline")
			return
		}
		byStage = append(byStage, s)
		totalCents += s.TotalCents
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not read pipeline")
		return
	}
	if byStage == nil {
		byStage = []stageSummary{}
	}

	var wonRate float64
	if decided := dealsWon + dealsLost; decided > 0 {
		wonRate = float64(dealsWon) / float64(decided)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"contacts_count":       contacts,
		"deals_open_count":     dealsOpen,
		"deals_won_count":      dealsWon,
		"deals_lost_count":     dealsLost,
		"tasks_pending_count":  tasksPending,
		"interactions_count":   interactions,
		"pipeline_by_stage":    byStage,
		"total_pipeline_cents": totalCents,
		"won_rate":             wonRate,
	})
}

type auditJSON struct {
	ID        int64           `json:"id"`
	UserID    *string         `json:"user_id,omitempty"`
	Action    string          `json:"action"`
	Entity    string          `json:"entity"`
	EntityID  *string         `json:"entity_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt string          `json:"created_at"`
}

// handleAuditList returns paginated audit logs, newest first.
func (s *Server) handleAuditList(w http.ResponseWriter, r *http.Request) {
	p := parsePagination(r)
	entity := strings.TrimSpace(r.URL.Query().Get("entity"))

	ctx := r.Context()
	database := s.db(r)

	var total int
	if err := database.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE org_id = $1 AND ($2 = '' OR entity = $2)`,
		OrgID(r), entity).Scan(&total); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list audit logs")
		return
	}

	rows, err := database.Query(ctx,
		`SELECT id, user_id, action, entity, entity_id, metadata,
		        to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM audit_logs
		 WHERE org_id = $1 AND ($2 = '' OR entity = $2)
		 ORDER BY id DESC LIMIT $3 OFFSET $4`,
		OrgID(r), entity, p.perPage, p.offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list audit logs")
		return
	}
	defer rows.Close()

	logs := []auditJSON{}
	for rows.Next() {
		var a auditJSON
		var meta string
		if err := rows.Scan(&a.ID, &a.UserID, &a.Action, &a.Entity, &a.EntityID, &meta, &a.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not read audit logs")
			return
		}
		a.Metadata = json.RawMessage(meta)
		logs = append(logs, a)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not read audit logs")
		return
	}
	writeJSON(w, http.StatusOK, pageEnvelope(logs, p, total))
}

type settingsJSON struct {
	OrgID     string         `json:"org_id"`
	Settings  map[string]any `json:"settings"`
	UpdatedAt string         `json:"updated_at"`
}

// handleSettingsGet returns the org settings document.
func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	var settings string
	var updated string
	err := s.db(r).QueryRow(r.Context(),
		`SELECT settings, to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM org_settings WHERE org_id = $1`, OrgID(r)).Scan(&settings, &updated)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not load settings")
		return
	}
	doc := map[string]any{}
	if err := json.Unmarshal([]byte(settings), &doc); err != nil {
		doc = map[string]any{}
	}
	writeJSON(w, http.StatusOK, settingsJSON{OrgID: OrgID(r), Settings: doc, UpdatedAt: updated})
}

type settingsPatchBody struct {
	Settings map[string]any `json:"settings"`
}

// handleSettingsPatch deep-merges the top level of the settings document.
func (s *Server) handleSettingsPatch(w http.ResponseWriter, r *http.Request) {
	var body settingsPatchBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Settings == nil {
		writeErr(w, http.StatusBadRequest, "invalid_input", "settings object is required")
		return
	}

	ctx := r.Context()
	database := s.db(r)

	var current string
	if err := database.QueryRow(ctx,
		`SELECT settings FROM org_settings WHERE org_id = $1`, OrgID(r)).Scan(&current); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not load settings")
		return
	}
	doc := map[string]any{}
	_ = json.Unmarshal([]byte(current), &doc)
	for k, v := range body.Settings {
		doc[k] = v
	}
	merged, err := json.Marshal(doc)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_input", "settings are not valid JSON")
		return
	}

	var updated string
	if err := database.QueryRow(ctx,
		`UPDATE org_settings SET settings = $2, updated_at = now()
		 WHERE org_id = $1
		 RETURNING to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`,
		OrgID(r), string(merged)).Scan(&updated); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not save settings")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "settings.updated", "settings", OrgID(r), body.Settings)
	writeJSON(w, http.StatusOK, settingsJSON{OrgID: OrgID(r), Settings: doc, UpdatedAt: updated})
}
