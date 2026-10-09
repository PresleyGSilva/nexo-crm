package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// DealStages enumerates the pipeline stages accepted by the API.
var DealStages = map[string]bool{
	"prospecting": true,
	"qualified":   true,
	"proposal":    true,
	"negotiation": true,
	"won":         true,
	"lost":        true,
}

type dealJSON struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   *string `json:"description,omitempty"`
	ContactID     *string `json:"contact_id,omitempty"`
	ValueCents    int64   `json:"value_cents"`
	Stage         string  `json:"stage"`
	ExpectedClose string  `json:"expected_close,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func scanDeal(columns func(dest ...any) error) (dealJSON, error) {
	var d dealJSON
	var created, updated time.Time
	var expected *time.Time
	err := columns(&d.ID, &d.Title, &d.Description, &d.ContactID, &d.ValueCents, &d.Stage, &expected, &created, &updated)
	if err != nil {
		return dealJSON{}, err
	}
	d.ExpectedClose = formatDate(expected)
	d.CreatedAt = created.UTC().Format(time.RFC3339)
	d.UpdatedAt = updated.UTC().Format(time.RFC3339)
	return d, nil
}

const dealColumns = `id, title, description, contact_id, value_cents, stage, expected_close, created_at, updated_at`

// handleDealsList returns paginated deals with optional stage/search filters.
func (s *Server) handleDealsList(w http.ResponseWriter, r *http.Request) {
	p := parsePagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	stage := strings.TrimSpace(r.URL.Query().Get("stage"))

	ctx := r.Context()
	database := s.db(r)

	var total int
	if err := database.QueryRow(ctx,
		`SELECT COUNT(*) FROM deals
		 WHERE org_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR stage = $3)`,
		OrgID(r), q, stage).Scan(&total); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list deals")
		return
	}

	rows, err := database.Query(ctx,
		`SELECT `+dealColumns+` FROM deals
		 WHERE org_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR stage = $3)
		 ORDER BY created_at DESC LIMIT $4 OFFSET $5`,
		OrgID(r), q, stage, p.perPage, p.offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list deals")
		return
	}
	defer rows.Close()

	deals := []dealJSON{}
	for rows.Next() {
		d, err := scanDeal(rows.Scan)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not read deals")
			return
		}
		deals = append(deals, d)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not read deals")
		return
	}
	writeJSON(w, http.StatusOK, pageEnvelope(deals, p, total))
}

type dealBody struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	ContactID     *string `json:"contact_id"`
	ValueCents    *int64  `json:"value_cents"`
	Stage         *string `json:"stage"`
	ExpectedClose *string `json:"expected_close"`
}

func resolveDealContact(body dealBody) (any, bool) {
	if body.ContactID == nil {
		return nil, true
	}
	id := strings.TrimSpace(*body.ContactID)
	if id == "" {
		return nil, true
	}
	if !isUUID(id) {
		return nil, false
	}
	return id, true
}

func resolveDealDate(body dealBody) (any, bool) {
	if body.ExpectedClose == nil {
		return nil, true
	}
	t, ok := parseDate(*body.ExpectedClose)
	if !ok {
		return nil, false
	}
	return t, true
}

// handleDealsCreate creates a deal inside the current org.
func (s *Server) handleDealsCreate(w http.ResponseWriter, r *http.Request) {
	var body dealBody
	if !decodeJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(strptr(body.Title))
	if title == "" {
		writeErr(w, http.StatusBadRequest, "invalid_input", "title is required")
		return
	}
	stage := strings.TrimSpace(strptr(body.Stage))
	if stage == "" {
		stage = "prospecting"
	}
	if !DealStages[stage] {
		writeErr(w, http.StatusBadRequest, "invalid_input", "unknown stage")
		return
	}
	contactID, ok := resolveDealContact(body)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_input", "contact_id must be a UUID")
		return
	}
	expected, ok := resolveDealDate(body)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_input", "expected_close must be YYYY-MM-DD")
		return
	}
	var value int64
	if body.ValueCents != nil {
		value = *body.ValueCents
	}

	ctx := r.Context()
	database := s.db(r)
	d, err := scanDeal(func(dest ...any) error {
		return database.QueryRow(ctx,
			`INSERT INTO deals (org_id, title, description, contact_id, value_cents, stage, expected_close, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING `+dealColumns,
			OrgID(r), title, cleanOpt(body.Description), contactID, value, stage, expected, UserID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create deal")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "deal.created", "deal", d.ID, map[string]string{"title": d.Title})
	writeJSON(w, http.StatusCreated, d)
}

// handleDealsGet returns a single org-scoped deal.
func (s *Server) handleDealsGet(w http.ResponseWriter, r *http.Request) {
	d, ok := s.loadDeal(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// handleDealsPatch partially updates an org-scoped deal.
// Only fields present in the body are touched; empty contact_id or
// expected_close clears the column.
func (s *Server) handleDealsPatch(w http.ResponseWriter, r *http.Request) {
	var body dealBody
	if !decodeJSON(w, r, &body) {
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "deal id must be a UUID")
		return
	}
	sets := []string{"updated_at = now()"}
	args := []any{id, OrgID(r)}
	push := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf(expr, len(args)))
	}
	if body.Title != nil && strings.TrimSpace(*body.Title) != "" {
		push("title = $%d", strings.TrimSpace(*body.Title))
	}
	if body.Description != nil {
		push("description = $%d", cleanOpt(body.Description))
	}
	if body.ContactID != nil {
		v := strings.TrimSpace(*body.ContactID)
		if v == "" {
			sets = append(sets, "contact_id = NULL")
		} else {
			if !isUUID(v) {
				writeErr(w, http.StatusBadRequest, "invalid_input", "contact_id must be a UUID")
				return
			}
			push("contact_id = $%d", v)
		}
	}
	if body.ValueCents != nil {
		push("value_cents = $%d", *body.ValueCents)
	}
	if body.Stage != nil {
		stage := strings.TrimSpace(*body.Stage)
		if !DealStages[stage] {
			writeErr(w, http.StatusBadRequest, "invalid_input", "unknown stage")
			return
		}
		push("stage = $%d", stage)
	}
	if body.ExpectedClose != nil {
		v := strings.TrimSpace(*body.ExpectedClose)
		if v == "" {
			sets = append(sets, "expected_close = NULL")
		} else {
			t, ok := parseDate(v)
			if !ok {
				writeErr(w, http.StatusBadRequest, "invalid_input", "expected_close must be YYYY-MM-DD")
				return
			}
			push("expected_close = $%d", t)
		}
	}

	ctx := r.Context()
	database := s.db(r)
	d, err := scanDeal(func(dest ...any) error {
		return database.QueryRow(ctx,
			`UPDATE deals SET `+strings.Join(sets, ", ")+`
			 WHERE id = $1 AND org_id = $2
			 RETURNING `+dealColumns, args...).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "deal not found")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "deal.updated", "deal", d.ID, nil)
	writeJSON(w, http.StatusOK, d)
}

type stageBody struct {
	Stage string `json:"stage"`
}

// handleDealsStage moves a deal to a new pipeline stage (audited).
func (s *Server) handleDealsStage(w http.ResponseWriter, r *http.Request) {
	var body stageBody
	if !decodeJSON(w, r, &body) {
		return
	}
	stage := strings.TrimSpace(body.Stage)
	if !DealStages[stage] {
		writeErr(w, http.StatusBadRequest, "invalid_input", "unknown stage")
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "deal id must be a UUID")
		return
	}

	ctx := r.Context()
	database := s.db(r)
	d, err := scanDeal(func(dest ...any) error {
		return database.QueryRow(ctx,
			`UPDATE deals SET stage = $3, updated_at = now()
			 WHERE id = $1 AND org_id = $2
			 RETURNING `+dealColumns, id, OrgID(r), stage).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "deal not found")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "deal.stage_changed", "deal", d.ID, map[string]string{"stage": stage})
	writeJSON(w, http.StatusOK, d)
}

// handleDealsDelete removes an org-scoped deal.
func (s *Server) handleDealsDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "deal id must be a UUID")
		return
	}
	tag, err := s.db(r).Exec(r.Context(),
		`DELETE FROM deals WHERE id = $1 AND org_id = $2`, id, OrgID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not delete deal")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "not_found", "deal not found")
		return
	}
	s.audit(r.Context(), s.db(r), OrgID(r), UserID(r), "deal.deleted", "deal", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadDeal(w http.ResponseWriter, r *http.Request, id string) (dealJSON, bool) {
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "deal id must be a UUID")
		return dealJSON{}, false
	}
	d, err := scanDeal(func(dest ...any) error {
		return s.db(r).QueryRow(r.Context(),
			`SELECT `+dealColumns+` FROM deals WHERE id = $1 AND org_id = $2`,
			id, OrgID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "deal not found")
		return dealJSON{}, false
	}
	return d, true
}
