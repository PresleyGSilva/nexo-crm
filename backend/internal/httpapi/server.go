// Package httpapi implements the REST API: routing, auth/tenant middleware
// and JSON handlers for every resource in the contract.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nexocrm/crm-saas/backend/internal/auth"
	"github.com/nexocrm/crm-saas/backend/internal/rbac"
)

// Server wires the connection pool and token stores to HTTP handlers.
type Server struct {
	Pool         *pgxpool.Pool
	JWTSecret    string
	AccessTTL    time.Duration
	Refresh      *auth.RefreshStore
	Resets       *auth.ResetStore
	CookieSecure bool
}

// NewServer builds a Server.
func NewServer(pool *pgxpool.Pool, secret string, accessTTL time.Duration, refresh *auth.RefreshStore, resets *auth.ResetStore, cookieSecure bool) *Server {
	return &Server{
		Pool:         pool,
		JWTSecret:    secret,
		AccessTTL:    accessTTL,
		Refresh:      refresh,
		Resets:       resets,
		CookieSecure: cookieSecure,
	}
}

type ctxKey string

const (
	userKey ctxKey = "user_id"
	orgKey  ctxKey = "org_id"
	roleKey ctxKey = "role"
	connKey ctxKey = "db_conn"
)

// DB is the query surface shared by *pgxpool.Pool, *pgxpool.Conn and pgx.Tx.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// db returns the request-scoped connection (with app.org_id set, so RLS
// applies) or the pool when no tenant connection is in context.
func (s *Server) db(r *http.Request) DB {
	if conn, ok := r.Context().Value(connKey).(*pgxpool.Conn); ok && conn != nil {
		return conn
	}
	return s.Pool
}

// UserID returns the authenticated user id from the request context.
func UserID(r *http.Request) string {
	v, _ := r.Context().Value(userKey).(string)
	return v
}

// OrgID returns the tenant org id from the request context.
func OrgID(r *http.Request) string {
	v, _ := r.Context().Value(orgKey).(string)
	return v
}

// UserRole returns the caller's role in the current org.
func UserRole(r *http.Request) rbac.Role {
	v, _ := r.Context().Value(roleKey).(rbac.Role)
	return v
}

// Router builds the full route tree for the API.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(slogMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(s.cors)

	r.Get("/healthz", s.handleHealth)
	r.Get("/openapi.yaml", s.handleOpenAPI)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/refresh", s.handleRefresh)
		r.Post("/auth/forgot", s.handleForgot)
		r.Post("/auth/reset", s.handleReset)

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/auth/me", s.handleMe)

			r.Group(func(r chi.Router) {
				r.Use(s.requireOrg)

				r.Get("/contacts", s.handleContactsList)
				r.With(s.requireRole(rbac.Member)).Post("/contacts", s.handleContactsCreate)
				r.Get("/contacts/{id}", s.handleContactsGet)
				r.With(s.requireRole(rbac.Member)).Patch("/contacts/{id}", s.handleContactsPatch)
				r.With(s.requireRole(rbac.Admin)).Delete("/contacts/{id}", s.handleContactsDelete)

				r.Get("/deals", s.handleDealsList)
				r.With(s.requireRole(rbac.Member)).Post("/deals", s.handleDealsCreate)
				r.Get("/deals/{id}", s.handleDealsGet)
				r.With(s.requireRole(rbac.Member)).Patch("/deals/{id}", s.handleDealsPatch)
				r.With(s.requireRole(rbac.Member)).Patch("/deals/{id}/stage", s.handleDealsStage)
				r.With(s.requireRole(rbac.Admin)).Delete("/deals/{id}", s.handleDealsDelete)

				r.Get("/tasks", s.handleTasksList)
				r.With(s.requireRole(rbac.Member)).Post("/tasks", s.handleTasksCreate)
				r.Get("/tasks/{id}", s.handleTasksGet)
				r.With(s.requireRole(rbac.Member)).Patch("/tasks/{id}", s.handleTasksPatch)
				r.With(s.requireRole(rbac.Admin)).Delete("/tasks/{id}", s.handleTasksDelete)

				r.Get("/interactions", s.handleInteractionsList)
				r.With(s.requireRole(rbac.Member)).Post("/interactions", s.handleInteractionsCreate)
				r.Get("/interactions/{id}", s.handleInteractionsGet)

				r.Get("/dashboard/summary", s.handleDashboardSummary)
				r.Get("/audit-logs", s.handleAuditList)

				r.Get("/settings", s.handleSettingsGet)
				r.With(s.requireRole(rbac.Admin)).Patch("/settings", s.handleSettingsPatch)
			})
		})
	})
	return r
}

// requireAuth validates the Bearer access token and stores the user id.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if header == "" || token == header || token == "" {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		userID, err := auth.ParseAccess(s.JWTSecret, token)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireOrg enforces tenant isolation: the caller must hold a membership in
// the org named by X-Org-ID. It then checks out a dedicated pool connection,
// sets app.org_id on it (so Postgres RLS policies apply to every query on
// this request) and stores the connection in the request context.
func (s *Server) requireOrg(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := strings.TrimSpace(r.Header.Get("X-Org-ID"))
		if orgID == "" {
			writeErr(w, http.StatusBadRequest, "org_required", "X-Org-ID header is required")
			return
		}
		if !isUUID(orgID) {
			writeErr(w, http.StatusBadRequest, "invalid_org", "X-Org-ID must be a UUID")
			return
		}
		var roleRaw string
		err := s.Pool.QueryRow(r.Context(),
			`SELECT role FROM memberships WHERE user_id = $1 AND org_id = $2`,
			UserID(r), orgID).Scan(&roleRaw)
		if err != nil {
			writeErr(w, http.StatusForbidden, "forbidden", "no access to organization")
			return
		}
		role, ok := rbac.Parse(roleRaw)
		if !ok {
			writeErr(w, http.StatusForbidden, "forbidden", "no access to organization")
			return
		}
		conn, err := s.Pool.Acquire(r.Context())
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, "db_unavailable", "could not acquire database connection")
			return
		}
		defer conn.Release()
		if _, err := conn.Exec(r.Context(), `SELECT set_config('app.org_id', $1, true)`, orgID); err != nil {
			writeErr(w, http.StatusInternalServerError, "db_error", "could not set tenant context")
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, orgKey, orgID)
		ctx = context.WithValue(ctx, roleKey, role)
		ctx = context.WithValue(ctx, connKey, conn)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRole rejects callers below the minimum role for an endpoint.
func (s *Server) requireRole(min rbac.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rbac.AtLeast(UserRole(r), min) {
				writeErr(w, http.StatusForbidden, "forbidden", "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// audit records an action in audit_logs. Failures are logged, never fatal.
func (s *Server) audit(ctx context.Context, database DB, orgID, userID, action, entity, entityID string, meta any) {
	var metaJSON []byte
	if meta != nil {
		var err error
		metaJSON, err = json.Marshal(meta)
		if err != nil {
			slog.Warn("audit marshal failed", "error", err)
			return
		}
	} else {
		metaJSON = []byte(`{}`)
	}
	var uid any
	if isUUID(userID) {
		uid = userID
	}
	if _, err := database.Exec(ctx,
		`INSERT INTO audit_logs (org_id, user_id, action, entity, entity_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		orgID, uid, action, entity, entityID, string(metaJSON)); err != nil {
		slog.Warn("audit insert failed", "error", err)
	}
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Org-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func slogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

type pageParams struct {
	page    int
	perPage int
	offset  int
}

func parsePagination(r *http.Request) pageParams {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return pageParams{page: page, perPage: perPage, offset: (page - 1) * perPage}
}

func pageEnvelope(data any, p pageParams, total int) map[string]any {
	return map[string]any{
		"data":     data,
		"page":     p.page,
		"per_page": p.perPage,
		"total":    total,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", "malformed JSON body")
		return false
	}
	return true
}
