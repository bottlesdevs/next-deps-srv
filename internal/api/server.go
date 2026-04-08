package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
	"github.com/bottlesdevs/next-deps-srv/internal/email"
	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/queue"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
)

type Server struct {
	store   *store.Store
	queue   *queue.BuildQueue
	backend bucket.Backend
	mailer  *email.Mailer
	secret  string
	dataDir string
}

func NewServer(s *store.Store, q *queue.BuildQueue, b bucket.Backend, mailer *email.Mailer, secret, dataDir string) *Server {
	return &Server{store: s, queue: q, backend: b, mailer: mailer, secret: secret, dataDir: dataDir}
}

func (srv *Server) Handler(rl *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()

	authMW := middleware.Auth(srv.store, srv.secret)
	rateMW := middleware.RateLimit(rl)
	contribMW := chain(authMW, middleware.RequireRole("admin", "mod", "contributor"))
	modMW := chain(authMW, middleware.RequireRole("admin", "mod"))
	adminMW := chain(authMW, middleware.RequireRole("admin"))

	distDir := filepath.Join(srv.dataDir, "..", "frontend", "dist")
	staticFS := http.FileServer(http.Dir(distDir))

	// SPA fallback: serve index.html for non-API, non-asset paths
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// let the file server try first
		path := filepath.Join(distDir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(path); err == nil {
			staticFS.ServeHTTP(w, r)
			return
		}
		// fallback to index.html for SPA routes
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	}))

	// avatars
	mux.Handle("GET /api/v1/users/{id}/avatar", http.HandlerFunc(srv.serveAvatar))

	// auth
	mux.Handle("POST /api/v1/auth/login", http.HandlerFunc(srv.login))
	mux.Handle("POST /api/v1/auth/register", http.HandlerFunc(srv.register))
	mux.Handle("GET /api/v1/auth/me", authMW(http.HandlerFunc(srv.getMe)))
	mux.Handle("PUT /api/v1/auth/me", authMW(http.HandlerFunc(srv.updateMe)))
	mux.Handle("POST /api/v1/auth/me/avatar", authMW(http.HandlerFunc(srv.uploadAvatar)))

	// deps (public read, rate-limited)
	mux.Handle("GET /api/v1/deps", rateMW(http.HandlerFunc(srv.listDeps)))
	mux.Handle("GET /api/v1/deps/{id}", rateMW(http.HandlerFunc(srv.getDep)))
	mux.Handle("POST /api/v1/deps", contribMW(http.HandlerFunc(srv.submitDep)))
	mux.Handle("GET /api/v1/deps/pending", modMW(http.HandlerFunc(srv.pendingDeps)))
	mux.Handle("POST /api/v1/deps/{id}/approve", modMW(http.HandlerFunc(srv.approveDep)))
	mux.Handle("POST /api/v1/deps/{id}/reject", modMW(http.HandlerFunc(srv.rejectDep)))

	// files
	mux.Handle("GET /api/v1/files/{name}", rateMW(http.HandlerFunc(srv.getFile)))
	mux.Handle("GET /api/v1/files/download/{rev_id}", rateMW(http.HandlerFunc(srv.downloadFile)))

	// community
	mux.Handle("GET /api/v1/community", authMW(http.HandlerFunc(srv.listPosts)))
	mux.Handle("POST /api/v1/community", authMW(http.HandlerFunc(srv.createPost)))
	mux.Handle("GET /api/v1/community/{id}/replies", authMW(http.HandlerFunc(srv.listReplies)))
	mux.Handle("POST /api/v1/community/{id}/replies", authMW(http.HandlerFunc(srv.createReply)))
	mux.Handle("DELETE /api/v1/community/{id}", authMW(http.HandlerFunc(srv.deletePost)))

	// admin
	mux.Handle("GET /api/v1/admin/stats", adminMW(http.HandlerFunc(srv.adminStats)))
	mux.Handle("GET /api/v1/admin/users", adminMW(http.HandlerFunc(srv.adminListUsers)))
	mux.Handle("GET /api/v1/admin/users/{id}", adminMW(http.HandlerFunc(srv.adminGetUser)))
	mux.Handle("PUT /api/v1/admin/users/{id}", adminMW(http.HandlerFunc(srv.adminUpdateUser)))
	mux.Handle("DELETE /api/v1/admin/users/{id}", adminMW(http.HandlerFunc(srv.adminDeleteUser)))
	mux.Handle("GET /api/v1/admin/jobs", adminMW(http.HandlerFunc(srv.adminListJobs)))
	mux.Handle("GET /api/v1/admin/jobs/{id}", adminMW(http.HandlerFunc(srv.adminGetJob)))
	mux.Handle("GET /api/v1/admin/jobs/{id}/log", adminMW(http.HandlerFunc(srv.adminJobLog)))
	mux.Handle("POST /api/v1/admin/jobs/{dep_id}/trigger", adminMW(http.HandlerFunc(srv.adminTriggerJob)))
	mux.Handle("GET /api/v1/admin/config", adminMW(http.HandlerFunc(srv.adminGetConfig)))
	mux.Handle("PUT /api/v1/admin/config", adminMW(http.HandlerFunc(srv.adminUpdateConfig)))
	mux.Handle("GET /api/v1/admin/audit", adminMW(http.HandlerFunc(srv.adminAudit)))
	mux.Handle("POST /api/v1/admin/backup", adminMW(http.HandlerFunc(srv.adminBackup)))
	mux.Handle("GET /api/v1/admin/backups", adminMW(http.HandlerFunc(srv.adminListBackups)))
	mux.Handle("POST /api/v1/admin/restore/{name}", adminMW(http.HandlerFunc(srv.adminRestore)))

	return mux
}

func chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func pageLimit(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func ipFrom(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	return r.RemoteAddr
}

func logAudit(ctx context.Context, s *store.Store, userID, username, action, resource, details, ip string) {
	_ = s.Log(ctx, models.AuditEntry{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ip,
	})
}
