package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// TaskStatuses enumerates the statuses accepted by the API.
var TaskStatuses = map[string]bool{
	"pending":     true,
	"in_progress": true,
	"done":        true,
	"cancelled":   true,
}

type taskJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
	DueDate     string  `json:"due_date,omitempty"`
	AssignedTo  *string `json:"assigned_to,omitempty"`
	ContactID   *string `json:"contact_id,omitempty"`
	DealID      *string `json:"deal_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func scanTask(columns func(dest ...any) error) (taskJSON, error) {
	var t taskJSON
	var created, updated time.Time
	var due *time.Time
	err := columns(&t.ID, &t.Title, &t.Description, &t.Status, &due, &t.AssignedTo, &t.ContactID, &t.DealID, &created, &updated)
	if err != nil {
		return taskJSON{}, err
	}
	t.DueDate = formatDate(due)
	t.CreatedAt = created.UTC().Format(time.RFC3339)
	t.UpdatedAt = updated.UTC().Format(time.RFC3339)
	return t, nil
}

const taskColumns = `id, title, description, status, due_date, assigned_to, contact_id, deal_id, created_at, updated_at`

// handleTasksList returns paginated tasks with optional status/search filters.
func (s *Server) handleTasksList(w http.ResponseWriter, r *http.Request) {
	p := parsePagination(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	ctx := r.Context()
	database := s.db(r)

	var total int
	if err := database.QueryRow(ctx,
		`SELECT COUNT(*) FROM tasks
		 WHERE org_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR status = $3)`,
		OrgID(r), q, status).Scan(&total); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list tasks")
		return
	}

	rows, err := database.Query(ctx,
		`SELECT `+taskColumns+` FROM tasks
		 WHERE org_id = $1
		   AND ($2 = '' OR title ILIKE '%' || $2 || '%')
		   AND ($3 = '' OR status = $3)
		 ORDER BY created_at DESC LIMIT $4 OFFSET $5`,
		OrgID(r), q, status, p.perPage, p.offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not list tasks")
		return
	}
	defer rows.Close()

	tasks := []taskJSON{}
	for rows.Next() {
		t, err := scanTask(rows.Scan)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not read tasks")
			return
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not read tasks")
		return
	}
	writeJSON(w, http.StatusOK, pageEnvelope(tasks, p, total))
}

type taskBody struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	DueDate     *string `json:"due_date"`
	AssignedTo  *string `json:"assigned_to"`
	ContactID   *string `json:"contact_id"`
	DealID      *string `json:"deal_id"`
}

func optUUID(raw *string) (any, bool) {
	if raw == nil {
		return nil, true
	}
	v := strings.TrimSpace(*raw)
	if v == "" {
		return nil, true
	}
	if !isUUID(v) {
		return nil, false
	}
	return v, true
}

// handleTasksCreate creates a task inside the current org.
func (s *Server) handleTasksCreate(w http.ResponseWriter, r *http.Request) {
	var body taskBody
	if !decodeJSON(w, r, &body) {
		return
	}
	title := strings.TrimSpace(strptr(body.Title))
	if title == "" {
		writeErr(w, http.StatusBadRequest, "invalid_input", "title is required")
		return
	}
	status := strings.TrimSpace(strptr(body.Status))
	if status == "" {
		status = "pending"
	}
	if !TaskStatuses[status] {
		writeErr(w, http.StatusBadRequest, "invalid_input", "unknown status")
		return
	}
	assigned, ok := optUUID(body.AssignedTo)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_input", "assigned_to must be a UUID")
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
	var due any
	if body.DueDate != nil && strings.TrimSpace(*body.DueDate) != "" {
		t, ok := parseDate(*body.DueDate)
		if !ok {
			writeErr(w, http.StatusBadRequest, "invalid_input", "due_date must be YYYY-MM-DD")
			return
		}
		due = t
	}

	ctx := r.Context()
	database := s.db(r)
	t, err := scanTask(func(dest ...any) error {
		return database.QueryRow(ctx,
			`INSERT INTO tasks (org_id, title, description, status, due_date, assigned_to, contact_id, deal_id, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING `+taskColumns,
			OrgID(r), title, cleanOpt(body.Description), status, due, assigned, contactID, dealID, UserID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create task")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "task.created", "task", t.ID, map[string]string{"title": t.Title})
	writeJSON(w, http.StatusCreated, t)
}

// handleTasksGet returns a single org-scoped task.
func (s *Server) handleTasksGet(w http.ResponseWriter, r *http.Request) {
	t, ok := s.loadTask(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handleTasksPatch partially updates an org-scoped task.
func (s *Server) handleTasksPatch(w http.ResponseWriter, r *http.Request) {
	var body taskBody
	if !decodeJSON(w, r, &body) {
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "task id must be a UUID")
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
	if body.Status != nil {
		status := strings.TrimSpace(*body.Status)
		if !TaskStatuses[status] {
			writeErr(w, http.StatusBadRequest, "invalid_input", "unknown status")
			return
		}
		push("status = $%d", status)
	}
	if body.DueDate != nil {
		v := strings.TrimSpace(*body.DueDate)
		if v == "" {
			sets = append(sets, "due_date = NULL")
		} else {
			t, ok := parseDate(v)
			if !ok {
				writeErr(w, http.StatusBadRequest, "invalid_input", "due_date must be YYYY-MM-DD")
				return
			}
			push("due_date = $%d", t)
		}
	}
	for _, field := range []struct {
		name string
		raw  *string
	}{
		{"assigned_to", body.AssignedTo},
		{"contact_id", body.ContactID},
		{"deal_id", body.DealID},
	} {
		if field.raw == nil {
			continue
		}
		v := strings.TrimSpace(*field.raw)
		if v == "" {
			sets = append(sets, field.name+" = NULL")
			continue
		}
		if !isUUID(v) {
			writeErr(w, http.StatusBadRequest, "invalid_input", field.name+" must be a UUID")
			return
		}
		args = append(args, v)
		sets = append(sets, fmt.Sprintf(field.name+" = $%d", len(args)))
	}

	ctx := r.Context()
	database := s.db(r)
	t, err := scanTask(func(dest ...any) error {
		return database.QueryRow(ctx,
			`UPDATE tasks SET `+strings.Join(sets, ", ")+`
			 WHERE id = $1 AND org_id = $2
			 RETURNING `+taskColumns, args...).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	s.audit(ctx, database, OrgID(r), UserID(r), "task.updated", "task", t.ID, nil)
	writeJSON(w, http.StatusOK, t)
}

// handleTasksDelete removes an org-scoped task.
func (s *Server) handleTasksDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "task id must be a UUID")
		return
	}
	tag, err := s.db(r).Exec(r.Context(),
		`DELETE FROM tasks WHERE id = $1 AND org_id = $2`, id, OrgID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not delete task")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	s.audit(r.Context(), s.db(r), OrgID(r), UserID(r), "task.deleted", "task", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadTask(w http.ResponseWriter, r *http.Request, id string) (taskJSON, bool) {
	if !isUUID(id) {
		writeErr(w, http.StatusBadRequest, "invalid_id", "task id must be a UUID")
		return taskJSON{}, false
	}
	t, err := scanTask(func(dest ...any) error {
		return s.db(r).QueryRow(r.Context(),
			`SELECT `+taskColumns+` FROM tasks WHERE id = $1 AND org_id = $2`,
			id, OrgID(r)).Scan(dest...)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "task not found")
		return taskJSON{}, false
	}
	return t, true
}
