package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nexocrm/crm-saas/backend/internal/auth"
	"github.com/nexocrm/crm-saas/backend/internal/rbac"
)

const refreshCookieName = "refresh_token"

type userJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type orgJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

func (s *Server) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(168 * time.Hour),
	})
}

func (s *Server) issueSession(w http.ResponseWriter, userID string) (string, bool) {
	access, err := auth.GenerateAccess(s.JWTSecret, userID, s.AccessTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token_error", "could not mint access token")
		return "", false
	}
	refresh, err := s.Refresh.Create(userID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token_error", "could not mint refresh token")
		return "", false
	}
	s.setRefreshCookie(w, refresh)
	return access, true
}

type registerBody struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	OrgName  string `json:"org_name"`
}

// handleRegister creates a user, a personal organization and an owner
// membership, then returns an authenticated session.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body registerBody
	if !decodeJSON(w, r, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	if body.Name == "" || !strings.Contains(body.Email, "@") || len(body.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "invalid_input", "name, valid email and password (min 8 chars) are required")
		return
	}

	ctx := r.Context()
	database := s.db(r)

	var exists bool
	if err := database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, body.Email).Scan(&exists); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not check user")
		return
	}
	if exists {
		writeErr(w, http.StatusConflict, "email_taken", "email already registered")
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash_error", "could not hash password")
		return
	}

	orgName := strings.TrimSpace(body.OrgName)
	if orgName == "" {
		prefix := body.Email
		if i := strings.Index(prefix, "@"); i > 0 {
			prefix = prefix[:i]
		}
		orgName = prefix + "'s workspace"
	}

	var user userJSON
	var createdAt time.Time
	if err := database.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, name, email, created_at`,
		body.Name, body.Email, hash).Scan(&user.ID, &user.Name, &user.Email, &createdAt); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create user")
		return
	}

	var org orgJSON
	if err := database.QueryRow(ctx,
		`INSERT INTO orgs (name, slug) VALUES ($1, $2) RETURNING id, name`,
		orgName, slugify(orgName)).Scan(&org.ID, &org.Name); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create organization")
		return
	}
	org.Role = string(rbac.Owner)

	if _, err := database.Exec(ctx,
		`INSERT INTO memberships (user_id, org_id, role) VALUES ($1, $2, 'owner')`,
		user.ID, org.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not create membership")
		return
	}
	if _, err := database.Exec(ctx,
		`INSERT INTO org_settings (org_id, settings) VALUES ($1, '{}') ON CONFLICT (org_id) DO NOTHING`,
		org.ID); err != nil {
		slog.Warn("settings init failed", "error", err)
	}

	s.audit(ctx, database, org.ID, user.ID, "org.created", "org", org.ID, map[string]string{"name": org.Name})

	access, ok := s.issueSession(w, user.ID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"access_token": access,
		"user":         user,
		"org":          org,
	})
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleLogin verifies credentials and returns a session plus the org list.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if !decodeJSON(w, r, &body) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))

	ctx := r.Context()
	database := s.db(r)

	var user userJSON
	var hash string
	if err := database.QueryRow(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = $1`,
		email).Scan(&user.ID, &user.Name, &user.Email, &hash); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err := auth.CheckPassword(hash, body.Password); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}

	orgs, err := s.userOrgs(ctx, database, user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not load memberships")
		return
	}

	access, ok := s.issueSession(w, user.ID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"user":         user,
		"orgs":         orgs,
	})
}

func (s *Server) userOrgs(ctx context.Context, database DB, userID string) ([]orgJSON, error) {
	return queryOrgs(ctx, database, userID)
}

// handleRefresh rotates the httpOnly refresh cookie into a new access token.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "missing refresh token")
		return
	}
	next, userID, ok := s.Refresh.Rotate(cookie.Value)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid or expired refresh token")
		return
	}
	access, err := auth.GenerateAccess(s.JWTSecret, userID, s.AccessTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token_error", "could not mint access token")
		return
	}
	s.setRefreshCookie(w, next)
	writeJSON(w, http.StatusOK, map[string]any{"access_token": access})
}

type forgotBody struct {
	Email string `json:"email"`
}

// handleForgot always succeeds to avoid account enumeration. When the email
// exists, a reset token is stored in memory and written to the server log
// (an e-mail provider plugs in here).
func (s *Server) handleForgot(w http.ResponseWriter, r *http.Request) {
	var body forgotBody
	if !decodeJSON(w, r, &body) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email != "" {
		var exists bool
		if err := s.db(r).QueryRow(r.Context(),
			`SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err == nil && exists {
			token, err := s.Resets.Create(email)
			if err == nil {
				slog.Info("password reset requested", "email", email, "reset_token", token)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "if the email exists, a reset link was sent"})
}

type resetBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// handleReset consumes a reset token and sets the new password.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	var body resetBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Token) == "" || len(body.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "invalid_input", "valid token and password (min 8 chars) are required")
		return
	}
	email, ok := s.Resets.Verify(body.Token)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_token", "reset token is invalid or expired")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash_error", "could not hash password")
		return
	}
	tag, err := s.db(r).Exec(r.Context(),
		`UPDATE users SET password_hash = $1, updated_at = now() WHERE email = $2`, hash, email)
	if err != nil || tag.RowsAffected() == 0 {
		writeErr(w, http.StatusBadRequest, "invalid_token", "reset token is invalid or expired")
		return
	}
	s.Resets.Invalidate(body.Token)
	s.Resets.InvalidateEmail(email)
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// handleMe returns the caller identity plus org memberships.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	database := s.db(r)
	var user userJSON
	if err := database.QueryRow(ctx,
		`SELECT id, name, email FROM users WHERE id = $1`, UserID(r)).Scan(&user.ID, &user.Name, &user.Email); err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	orgs, err := queryOrgs(ctx, database, user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db_error", "could not load memberships")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "orgs": orgs})
}
