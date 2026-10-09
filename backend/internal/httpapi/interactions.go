package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// InteractionTypes enumerates the interaction channels accepted by the API.
var InteractionTypes = map[string]bool{
	"call":     true,
	"email":    true,
	"meeting":  true,
	"note":     true,
	"whatsapp": true,
}

type interactionJSON struct {
	ID        string  `json:"id"`
	ContactID *string `json:"contact_id,omitempty"`
	DealID    *string `json:"deal_id,omitempty"`
	Type      string  `json:"type"`
	Content   string  `json:"content"`
	CreatedAt string  `json:"created_at"`
}

func scanInteraction(columns func(dest ...any) error) (interactionJSON, error) {
	var it interactionJSON
	var created time.Time
	err := columns(&it.ID, &it.ContactID, &it.DealID, &it.Type, &it.Content, &created)
	if err != nil {
		return interactionJSON{}, err
	}
	it.CreatedAt = created.UTC().Format(time.RFC3339)
	return it, nil
}

const interactionColumns = `id, contact_id, deal_id, type, content, created_at`

// handleInteractionsList returns paginated interactions, optionally scoped to
// a contact or deal.
func (s *Server) handleInteractionsList(w http.ResponseWriter, r *http.Request) {
	p := parsePagination(r)
	contactID := strings.TrimSpace(r.URL.Query().Get("contact_id"))
	dealID := strings.TrimSpace(r.URL.Query().Get("deal_id"))
	if contactID != "" && !isUUID(contactID) {
		writeErr(w, http.StatusBadRequest, "invalid_input", "contact_id must be a UUID")
		return
	}
	if dealID != "" && !isUUID(dealID) {
		writeErr(w, http.StatusBadRequest, "invalid_input", "deal_id must be a UUID")
		return
	}
	ctx := r.Context()
	database := s.db(r)

	var contactFilter, dealFilter any
	if contactID != "" {
		contactFilter = contactID
	}
	if dealID != "" {
		dealFilter = dealID
	}

	var total int
	if err := database.QueryRow(ctx,
		`SELECT COUNT(*) FROM interactions
		 WHERE org_id = $1
		   AND ($2::uuid IS NULL OR contact_id = $2::uuid)
		   AND ($3::uuid IS NULL OR deal_id = $3::uuid)`,
		OrgID(r), contactFilter, dealFilter).Scan(&total); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list interactions")
		return
	}

	rows, err := database.Query(ctx,
		`SELECT `+interactionColumns+` FROM interactions
		 WHERE org_id = $1
		   AND ($2::uuid IS NULL OR contact_id = $2::uuid)
		   AND ($3::uuid IS NULL OR deal_id = $3::uuid)
		 ORDER BY created_at DESC LIMIT $4 OFFSET $5`,
		OrgID(r), contactFilter, dealFilter, p.perPage, p.offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list interactions")
		return
	}
	defer rows.Close()

	items := []interactionJSON{}
	for rows.Next() {
		it, err := scanInteraction(rows.Scan)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not read interactions")
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not read interactions")
		return
	}
	writeJSON(w, http.StatusOK, pageEnvelope(items, p, total))
}

type interactionBody struct {
	ContactID *string `json:"contact_id"`
	DealID    *string `json:"deal_id"`
	Type      *string `json:"type"`
	Content   *string `json:"content"`
}

// handleInteractionsCreate records an interaction inside the current org.
func (s *Server) handleInteractionsCreate(w http.ResponseWriter, r *http.Request) {
	var body interactionBody
	if !decodeJSON(w, r, &body) {
		return
	}
	kind := strings.ToLower(strings.TrimSpace(strptr(body.Type)))
	if !InteractionTypes[kind] {
		writeErr(w, http.StatusBadRequest, "invalid_input", "type must be one of: call, email, meeting, note, whatsapp")
		return
	}
	content := strings.TrimSpace(strptr(body.Content))
	if content == "" {
		writeErr(w, http.StatusBadRequest, "invalid_input", "content is required")
		return
	}
	contactID, ok := optUUID(body.ContactID)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_input", "contact_id must be a UUID")
		return
	}
	dealID, ok := optUUID(body.DealID)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_input", "deal_id must be a UUID")
		return
	}
	if contactID == nil && dealID == nil {
		writeErr(w, http.StatusBadRequest, "invalid_input", "contact_id or deal_id is required")
		return
	}

	ctx := r.Context()
	database := s.db(r)
	it, err := scanInteraction(func(dest ...any) error {
		return database.QueryRow(ctx,
			`INSERT INTO interactions (org_id, contact_id, deal_id, type, content, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING `+interactionColumns,
			OrgID(r), contactID, dealID, kind, content, UserID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create interaction")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "interaction.created", "interaction", it.ID, map[string]string{"type": kind})
	writeJSON(w, http.StatusCreated, it)
}

// handleInteractionsGet returns a single org-scoped interaction.
func (s *Server) handleInteractionsGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "interaction id must be a UUID")
		return
	}
	it, err := scanInteraction(func(dest ...any) error {
		return s.db(r).QueryRow(r.Context(),
			`SELECT `+interactionColumns+` FROM interactions WHERE id = $1 AND org_id = $2`,
			id, OrgID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "interaction not found")
		return
	}
	writeJSON(w, http.StatusOK, it)
}
