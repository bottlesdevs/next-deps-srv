package api

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

func (srv *Server) adminStats(w http.ResponseWriter, r *http.Request) {
	users, _ := srv.store.ListUsers(r.Context())
	deps, _ := srv.store.ListAllDeps(r.Context())
	jobs, _ := srv.store.ListJobs(r.Context())

	var pendingDeps, builtDeps int
	for _, d := range deps {
		if d.Status == "pending_review" {
			pendingDeps++
		} else if d.Status == "built" {
			builtDeps++
		}
	}
	var runningJobs, failedJobs int
	for _, j := range jobs {
		if j.Status == "running" {
			runningJobs++
		} else if j.Status == "failed" {
			failedJobs++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users":        len(users),
		"deps_total":   len(deps),
		"deps_pending": pendingDeps,
		"deps_built":   builtDeps,
		"jobs_total":   len(jobs),
		"jobs_running": runningJobs,
		"jobs_failed":  failedJobs,
	})
}

func (srv *Server) adminListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	users, err := srv.store.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	var out []safeUserOut
	for _, u := range users {
		out = append(out, safeUser(u))
	}
	writeJSON(w, http.StatusOK, paginate(out, page, limit))
}

func (srv *Server) adminGetUser(w http.ResponseWriter, r *http.Request) {
	user, err := srv.store.GetUser(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, safeUser(user))
}

func (srv *Server) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	user, err := srv.store.GetUser(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	var body struct {
		Roles   []string `json:"roles"`
		Email   string   `json:"email"`
		Enabled *bool    `json:"enabled"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	oldRoles := user.Roles
	if body.Roles != nil {
		user.Roles = body.Roles
	}
	if body.Email != "" {
		user.Email = body.Email
	}
	if body.Enabled != nil {
		user.Enabled = *body.Enabled
	}
	user.UpdatedAt = time.Now()
	if err := srv.store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "admin_update_user", user.ID, user.Username, ipFrom(r))
	if srv.mailer != nil && body.Roles != nil && rolesChanged(oldRoles, user.Roles) {
		u := user
		go func() {
			if err := srv.mailer.RoleChanged(u, u.Roles); err != nil {
				log.Printf("mail: %v", err)
			}
		}()
	}
	writeJSON(w, http.StatusOK, safeUser(user))
}

func (srv *Server) adminDeleteUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	id := r.PathValue("id")
	if id == claims.UserID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete self"})
		return
	}
	if err := srv.store.DeleteUser(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "admin_delete_user", id, "", ipFrom(r))
	w.WriteHeader(http.StatusNoContent)
}

func (srv *Server) adminListJobs(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	jobs, err := srv.store.ListJobs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusOK, paginate(jobs, page, limit))
}

func (srv *Server) adminGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := srv.store.GetJob(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (srv *Server) adminJobLog(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := srv.queue.Subscribe(jobID)
	for {
		select {
		case <-r.Context().Done():
			return
		case line, open := <-ch:
			if !open {
				fmt.Fprintf(w, "event: done\ndata: \n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

func (srv *Server) adminTriggerJob(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	depID := r.PathValue("dep_id")
	dep, err := srv.store.GetDep(r.Context(), depID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "dep not found"})
		return
	}
	job, err := srv.queue.Submit(r.Context(), dep)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "queue error"})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "trigger_job", dep.ID, dep.Name, ipFrom(r))
	writeJSON(w, http.StatusCreated, job)
}

func (srv *Server) adminGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := srv.store.GetConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (srv *Server) adminUpdateConfig(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	var cfg models.AppConfig
	if err := readJSON(r, &cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := srv.store.SaveConfig(r.Context(), cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save failed"})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "update_config", "config", "", ipFrom(r))
	writeJSON(w, http.StatusOK, cfg)
}

func (srv *Server) adminAudit(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	entries, err := srv.store.ListAudit(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store error"})
		return
	}
	writeJSON(w, http.StatusOK, paginate(entries, page, limit))
}

func (srv *Server) adminBackup(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	backupDir := filepath.Join(srv.dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mkdir error"})
		return
	}
	name := fmt.Sprintf("%d.zip", time.Now().Unix())
	dest := filepath.Join(backupDir, name)
	if err := zipDir(srv.dataDir, dest, "backups"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "backup error: " + err.Error()})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "backup", name, "", ipFrom(r))
	writeJSON(w, http.StatusCreated, map[string]string{"name": name})
}

func (srv *Server) adminListBackups(w http.ResponseWriter, r *http.Request) {
	backupDir := filepath.Join(srv.dataDir, "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".zip") {
			names = append(names, e.Name())
		}
	}
	writeJSON(w, http.StatusOK, names)
}

func (srv *Server) adminRestore(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFrom(r)
	name := r.PathValue("name")
	if strings.Contains(name, "..") || !strings.HasSuffix(name, ".zip") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid name"})
		return
	}
	archivePath := filepath.Join(srv.dataDir, "backups", name)
	if _, err := os.Stat(archivePath); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "backup not found"})
		return
	}
	if err := unzipRestore(archivePath, srv.dataDir); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "restore error: " + err.Error()})
		return
	}
	logAudit(r.Context(), srv.store, claims.UserID, claims.Username, "restore", name, "", ipFrom(r))
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func zipDir(src, dest, skipSubdir string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if info.IsDir() {
			if rel == skipSubdir || strings.HasPrefix(rel, skipSubdir+string(os.PathSeparator)) {
				return filepath.SkipDir
			}
			return nil
		}
		fw, err := zw.Create(rel)
		if err != nil {
			return err
		}
		rf, err := os.Open(path)
		if err != nil {
			return err
		}
		defer rf.Close()
		_, err = io.Copy(fw, rf)
		return err
	})
}

func unzipRestore(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		target := filepath.Join(dest, filepath.Clean(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("zip-slip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func rolesChanged(a, b []string) bool {
	if len(a) != len(b) {
		return true
	}
	set := make(map[string]struct{}, len(a))
	for _, r := range a {
		set[r] = struct{}{}
	}
	for _, r := range b {
		if _, ok := set[r]; !ok {
			return true
		}
	}
	return false
}
