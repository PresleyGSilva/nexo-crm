package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type contactJSON struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Company   *string `json:"company,omitempty"`
	Position  *string `json:"position,omitempty"`
	Status    string  `json:"status"`
	Notes     *string `json:"notes,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func scanContact(columns func(dest ...any) error) (contactJSON, error) {
	var c contactJSON
	var created, updated time.Time
	err := columns(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Company, &c.Position, &c.Status, &c.Notes, &created, &updated)
	if err != nil {
		return contactJSON{}, err
	}
	c.CreatedAt = created.UTC().Format(time.RFC3339)
	c.UpdatedAt = updated.UTC().Format(time.RFC3339)
	return c, nil
}

const contactColumns = `id, name, email, phone, company, position, status, notes, created_at, updated_at`

// handleContactsList returns paginated contacts with optional search/filter.
func (s *Server) handleContactsList(w http.ResponseWriter, r *http.Request) {
	p := parsePagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	ctx := r.Context()
	database := s.db(r)

	var total int
	if err := database.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts
		 WHERE org_id = $1
		   AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%' OR company ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR status = $3)`,
		OrgID(r), q, status).Scan(&total); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list contacts")
		return
	}

	rows, err := database.Query(ctx,
		`SELECT `+contactColumns+` FROM contacts
		 WHERE org_id = $1
		   AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR email ILIKE '%' || $2 || '%' OR company ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR status = $3)
		 ORDER BY created_at DESC LIMIT $4 OFFSET $5`,
		OrgID(r), q, status, p.perPage, p.offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list contacts")
		return
	}
	defer rows.Close()

	contacts := []contactJSON{}
	for rows.Next() {
		c, err := scanContact(rows.Scan)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not read contacts")
			return
		}
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not read contacts")
		return
	}
	writeJSON(w, http.StatusOK, pageEnvelope(contacts, p, total))
}

type contactBody struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Company  *string `json:"company"`
	Position *string `json:"position"`
	Status   *string `json:"status"`
	Notes    *string `json:"notes"`
}

func cleanOpt(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// handleContactsCreate creates a contact inside the current org.
func (s *Server) handleContactsCreate(w http.ResponseWriter, r *http.Request) {
	var body contactBody
	if !decodeJSON(w, r, &body) {
		return
	}
	name := ""
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
	}
	if name == "" {
		writeErr(w, http.StatusBadRequest, "invalid_input", "name is required")
		return
	}
	status := strptr(body.Status)
	if status == "" {
		status = "active"
	}

	ctx := r.Context()
	database := s.db(r)
	c, err := scanContact(func(dest ...any) error {
		return database.QueryRow(ctx,
			`INSERT INTO contacts (org_id, name, email, phone, company, position, status, notes, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING `+contactColumns,
			OrgID(r), name, cleanOpt(body.Email), cleanOpt(body.Phone), cleanOpt(body.Company),
			cleanOpt(body.Position), status, cleanOpt(body.Notes), UserID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create contact")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "contact.created", "contact", c.ID, map[string]string{"name": c.Name})
	writeJSON(w, http.StatusCreated, c)
}

// handleContactsGet returns a single org-scoped contact.
func (s *Server) handleContactsGet(w http.ResponseWriter, r *http.Request) {
	c, ok := s.loadContact(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// handleContactsPatch partially updates an org-scoped contact.
func (s *Server) handleContactsPatch(w http.ResponseWriter, r *http.Request) {
	var body contactBody
	if !decodeJSON(w, r, &body) {
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "contact id must be a UUID")
		return
	}
	status := ""
	if body.Status != nil {
		status = strings.TrimSpace(*body.Status)
	}

	ctx := r.Context()
	database := s.db(r)
	c, err := scanContact(func(dest ...any) error {
		return database.QueryRow(ctx,
			`UPDATE contacts SET
				name = COALESCE(NULLIF($3, ''), name),
				email = COALESCE($4, email),
				phone = COALESCE($5, phone),
				company = COALESCE($6, company),
				position = COALESCE($7, position),
				status = COALESCE(NULLIF($8, ''), status),
				notes = COALESCE($9, notes),
				updated_at = now()
			 WHERE id = $1 AND org_id = $2
			 RETURNING `+contactColumns,
			id, OrgID(r),
			strptr(body.Name), cleanOpt(body.Email), cleanOpt(body.Phone),
			cleanOpt(body.Company), cleanOpt(body.Position), status, body.Notes).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "contact not found")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "contact.updated", "contact", c.ID, nil)
	writeJSON(w, http.StatusOK, c)
}

// handleContactsDelete removes an org-scoped contact.
func (s *Server) handleContactsDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "contact id must be a UUID")
		return
	}
	tag, err := s.db(r).Exec(r.Context(),
		`DELETE FROM contacts WHERE id = $1 AND org_id = $2`, id, OrgID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not delete contact")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "not_found", "contact not found")
		return
	}
	s.audit(r.Context(), s.db(r), OrgID(r), UserID(r), "contact.deleted", "contact", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadContact(w http.ResponseWriter, r *http.Request, id string) (contactJSON, bool) {
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "contact id must be a UUID")
		return contactJSON{}, false
	}
	c, err := scanContact(func(dest ...any) error {
		return s.db(r).QueryRow(r.Context(),
			`SELECT `+contactColumns+` FROM contacts WHERE id = $1 AND org_id = $2`,
			id, OrgID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "contact not found")
		return contactJSON{}, false
	}
	return c, true
}
